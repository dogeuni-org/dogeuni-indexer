package explorer

import (
	"database/sql/driver"
	"dogeuni-indexer/models"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/rpcclient"
)

func mintTx() *btcjson.TxRawResult {
	return &btcjson.TxRawResult{
		Hash: "h1", Txid: "h1",
		Vin:  []btcjson.Vin{{Txid: "0000000000000000000000000000000000000000000000000000000000000001"}},
		Vout: []btcjson.Vout{{Value: 0.001, ScriptPubKey: btcjson.ScriptPubKeyResult{Addresses: []string{"DHolder"}}}},
	}
}

const mintJSON = `{"p":"drc-20","op":"mint","tick":"TEST","amt":"1"}`

// a node failure must abort the block for a retry, never drop the inscription
func TestDecodeNodeErrorIsRetryable(t *testing.T) {
	e := newTestExplorer(t, &models.Drc20Info{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "node unavailable", http.StatusInternalServerError)
	}))
	defer srv.Close()
	node, err := rpcclient.New(&rpcclient.ConnConfig{Host: strings.TrimPrefix(srv.URL, "http://"), User: "u", Pass: "p", HTTPPostMode: true, DisableTLS: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	e.node = node

	_, err = e.drc20Decode(mintTx(), []byte(mintJSON), 1)
	if err == nil || !retryable(err) {
		t.Fatalf("want retryable node error, got %v", err)
	}

	var n int64
	e.dbc.DB.Model(&models.Drc20Info{}).Count(&n)
	if n != 0 {
		t.Fatalf("inscription saved despite node error: %d rows", n)
	}
}

func TestDecodeDedup(t *testing.T) {
	e := newTestExplorer(t, &models.Drc20Info{})
	if err := e.dbc.DB.Create(&models.Drc20Info{TxHash: "h1"}).Error; err != nil {
		t.Fatal(err)
	}

	// an already indexed tx is skipped, not retried
	_, err := e.drc20Decode(mintTx(), []byte(mintJSON), 1)
	if err == nil || retryable(err) {
		t.Fatalf("want non-retryable duplicate error, got %v", err)
	}

	// a failing dedup lookup is a database fault, not a duplicate: the scanner aborts the block
	on := true
	injectQueryFault(t, e, &on)
	_, err = e.drc20Decode(mintTx(), []byte(mintJSON), 1)
	on = false
	if err == nil {
		t.Fatal("want dedup error under query fault")
	}
	if fault := e.fault.Take(); !errors.Is(fault, driver.ErrBadConn) {
		t.Fatalf("dedup fault not recorded, got %v", fault)
	}
}

func TestInvalidInscriptionNotRetryable(t *testing.T) {
	e := newTestExplorer(t, &models.Drc20Info{})
	tx := mintTx()
	tx.Vout[0].ScriptPubKey.Addresses = nil
	_, err := e.drc20Decode(tx, []byte(mintJSON), 1)
	if err == nil || retryable(err) {
		t.Fatalf("want non-retryable invalid-inscription error, got %v", err)
	}
}
