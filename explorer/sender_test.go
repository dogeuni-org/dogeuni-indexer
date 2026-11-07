package explorer

import (
	"database/sql/driver"
	"dogeuni-indexer/models"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/btcutil"
	"github.com/dogecoinw/doged/chaincfg"
	"github.com/dogecoinw/doged/rpcclient"
)

const (
	prevTxid   = "00000000000000000000000000000000000000000000000000000000000000aa"
	commitTxid = "00000000000000000000000000000000000000000000000000000000000000bb"
)

// fakeNode serves getrawtransaction from txs.
func fakeNode(t *testing.T, txs map[string]*btcjson.TxRawResult) *rpcclient.Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Params) == 0 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var txid string
		json.Unmarshal(req.Params[0], &txid)
		res := map[string]interface{}{"id": req.ID, "result": nil, "error": nil}
		if tx, ok := txs[txid]; ok {
			res["result"] = tx
		} else {
			res["error"] = map[string]interface{}{"code": -5, "message": "No such transaction"}
		}
		json.NewEncoder(w).Encode(res)
	}))
	t.Cleanup(srv.Close)
	node, err := rpcclient.New(&rpcclient.ConnConfig{Host: strings.TrimPrefix(srv.URL, "http://"), User: "u", Pass: "p", HTTPPostMode: true, DisableTLS: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return node
}

func p2sh(t *testing.T, script string) string {
	a, err := btcutil.NewAddressScriptHash([]byte(script), &chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	return a.String()
}

func out(addr string) btcjson.Vout {
	return btcjson.Vout{Value: 0.001, ScriptPubKey: btcjson.ScriptPubKeyResult{Addresses: []string{addr}}}
}

// senderChain returns a commit tx whose first input spends an output of sender.
func senderChain(t *testing.T, e *Explorer, sender string) *btcjson.TxRawResult {
	commit := &btcjson.TxRawResult{Txid: commitTxid, Hash: commitTxid,
		Vin: []btcjson.Vin{{Txid: prevTxid, Vout: 0}}, Vout: []btcjson.Vout{out("DFee")}}
	e.node = fakeNode(t, map[string]*btcjson.TxRawResult{
		prevTxid:   {Txid: prevTxid, Hash: prevTxid, Vout: []btcjson.Vout{out(sender)}},
		commitTxid: commit,
	})
	return commit
}

// every table in reserveModels must be found by its reserves_address
func TestCommitSenderRejectsEveryReserveKind(t *testing.T) {
	for _, m := range reserveModels {
		e := newTestExplorer(t, reserveModels...)
		e.reserveSenderHeight = 100
		reserve := p2sh(t, "WOOWOW")
		if err := e.dbc.DB.Model(m).Create(map[string]interface{}{"reserves_address": reserve}).Error; err != nil {
			t.Fatalf("%T: %v", m, err)
		}
		commit := senderChain(t, e, reserve)

		if _, err := e.commitSender(commit, 100); err == nil || retryable(err) {
			t.Fatalf("%T: want non-retryable reserve rejection, got %v", m, err)
		}
		// before the activation height the legacy rule still holds
		if got, err := e.commitSender(commit, 99); err != nil || got != reserve {
			t.Fatalf("%T: legacy sender = %q, %v", m, got, err)
		}
	}
}

func TestCommitSenderAllowsOrdinaryAddresses(t *testing.T) {
	e := newTestExplorer(t, reserveModels...)
	e.reserveSenderHeight = 100
	e.dbc.DB.Create(&models.SwapLiquidity{ReservesAddress: p2sh(t, "WOOWOW")})

	for _, sender := range []string{
		"DHolderPlainP2PKH", // not P2SH: no lookup at all
		p2sh(t, "multisig"), // P2SH but not a reserve
	} {
		commit := senderChain(t, e, sender)
		if got, err := e.commitSender(commit, 100); err != nil || got != sender {
			t.Fatalf("sender %s: got %q, %v", sender, got, err)
		}
	}
}

// a DRC-20 transfer that spends a reserve's UTXO is not indexed
func TestDrc20TransferFromReserveRejected(t *testing.T) {
	e := newTestExplorer(t, append([]interface{}{&models.Drc20Info{}}, reserveModels...)...)
	e.reserveSenderHeight = 100
	reserve := p2sh(t, "WOOWOW")
	e.dbc.DB.Create(&models.SwapLiquidity{ReservesAddress: reserve})
	senderChain(t, e, reserve)

	reveal := &btcjson.TxRawResult{Txid: "r1", Hash: "r1",
		Vin: []btcjson.Vin{{Txid: commitTxid, Vout: 0}}, Vout: []btcjson.Vout{out("DAttacker")}}
	transfer := []byte(`{"p":"drc-20","op":"transfer","tick":"WOO","amt":"1"}`)

	if _, err := e.drc20Decode(reveal, transfer, 100); err == nil || retryable(err) {
		t.Fatalf("want non-retryable rejection, got %v", err)
	}
	var n int64
	e.dbc.DB.Model(&models.Drc20Info{}).Count(&n)
	if n != 0 {
		t.Fatalf("transfer from a reserve was saved: %d rows", n)
	}

	// legacy height: the same transfer decodes with the reserve as sender
	card, err := e.drc20Decode(reveal, transfer, 99)
	if err != nil || card.HolderAddress != reserve {
		t.Fatalf("legacy decode = %+v, %v", card, err)
	}
}

// a failing reserve lookup aborts the block instead of letting the tx through
func TestCommitSenderLookupFaultIsRecorded(t *testing.T) {
	e := newTestExplorer(t, reserveModels...)
	e.reserveSenderHeight = 100
	commit := senderChain(t, e, p2sh(t, "WOOWOW"))
	on := true
	injectQueryFault(t, e, &on)
	_, err := e.commitSender(commit, 100)
	on = false
	if err == nil {
		t.Fatal("want lookup error")
	}
	if fault := e.fault.Take(); !errors.Is(fault, driver.ErrBadConn) {
		t.Fatalf("lookup fault not recorded, got %v", fault)
	}
}

// reserveModels must list every model that records a reserve address, so a new
// protocol's reserve cannot be left out of the sender check
func TestReserveModelsCoverEveryReserveTable(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "../models", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						if name.Name == "ReservesAddress" {
							want[ts.Name.Name] = true
						}
					}
				}
				return true
			})
		}
	}

	got := map[string]bool{}
	for _, m := range reserveModels {
		got[reflect.TypeOf(m).Elem().Name()] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("models.%s has ReservesAddress but is missing from reserveModels", name)
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("reserveModels lists %s, which has no ReservesAddress", name)
		}
	}
}
