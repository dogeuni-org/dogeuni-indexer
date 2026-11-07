package router

import (
	"bytes"
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func drc20Post(t *testing.T, handler gin.HandlerFunc, body string) (int, []map[string]interface{}) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/", handler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(body))
	engine.ServeHTTP(w, req)
	out := struct {
		Data []map[string]interface{} `json:"data"`
	}{}
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
	}
	return w.Code, out.Data
}

func seedTransfer(t *testing.T, dbc *storage.DBClient, hash, to string, height, status int64) {
	info := &models.Drc20Info{Op: "transfer", Tick: "TICK", Amt: models.NewNumber(10), ToAddress: to,
		TxHash: hash, BlockNumber: height, OrderStatus: status}
	if err := dbc.DB.Create(info).Error; err != nil {
		t.Fatal(err)
	}
	key := &models.Drc20TransferKey{TxHash: hash, BlockNumber: height, SenderPubkey: "key-" + hash,
		SenderTxid: "prev-" + hash, SenderVout: 3}
	if err := dbc.DB.Create(key).Error; err != nil {
		t.Fatal(err)
	}
}

func TestTransferScan(t *testing.T) {
	dbc := newTestDB(t, &models.Drc20Info{}, &models.Drc20TransferKey{})
	seedTransfer(t, dbc, "a", "A", 100, 0)
	seedTransfer(t, dbc, "failed", "F", 101, 1)
	seedTransfer(t, dbc, "b", "B,C", 102, 0)
	handler := (&Drc20Router{dbc: dbc}).TransferScan

	code, rows := drc20Post(t, handler, `{}`)
	if code != http.StatusOK || len(rows) != 2 {
		t.Fatalf("code %d rows %v", code, rows)
	}
	if rows[0]["tx_hash"] != "a" || rows[0]["sender_pubkey"] != "key-a" || rows[0]["tick"] != "TICK" {
		t.Fatalf("row %v", rows[0])
	}
	if rows[0]["sender_txid"] != "prev-a" || rows[0]["sender_vout"] != float64(3) {
		t.Fatalf("outpoint %v", rows[0])
	}
	if to := rows[1]["to_addresses"].([]interface{}); len(to) != 2 || to[1] != "C" {
		t.Fatalf("to_addresses %v", to)
	}

	_, next := drc20Post(t, handler, fmt.Sprintf(`{"after_id":%v}`, rows[0]["id"]))
	if len(next) != 1 || next[0]["tx_hash"] != "b" {
		t.Fatalf("after_id rows %v", next)
	}
	_, from := drc20Post(t, handler, `{"from_block":102}`)
	if len(from) != 1 || from[0]["tx_hash"] != "b" {
		t.Fatalf("from_block rows %v", from)
	}
	_, one := drc20Post(t, handler, `{"limit":1}`)
	if len(one) != 1 || one[0]["tx_hash"] != "a" {
		t.Fatalf("limit rows %v", one)
	}
}

func TestBalances(t *testing.T) {
	dbc := newTestDB(t, &models.Drc20CollectAddress{})
	for _, row := range []models.Drc20CollectAddress{
		{Tick: "AAA", HolderAddress: "H1", AmtSum: models.NewNumber(5)},
		{Tick: "BBB", HolderAddress: "H1", AmtSum: models.NewNumber(7)},
		{Tick: "AAA", HolderAddress: "H2", AmtSum: models.NewNumber(0)},
		{Tick: "AAA", HolderAddress: "H3", AmtSum: models.NewNumber(9)},
	} {
		row := row
		if err := dbc.DB.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	handler := (&Drc20Router{dbc: dbc}).Balances

	code, rows := drc20Post(t, handler, `{"addresses":["H1","H2"]}`)
	if code != http.StatusOK || len(rows) != 2 {
		t.Fatalf("code %d rows %v", code, rows)
	}
	if rows[0]["tick"] != "AAA" || rows[0]["holder_address"] != "H1" || fmt.Sprint(rows[0]["amt"]) != "5" {
		t.Fatalf("row %v", rows[0])
	}
	if _, rows = drc20Post(t, handler, `{"addresses":["H1"],"tick":"BBB"}`); len(rows) != 1 || rows[0]["tick"] != "BBB" {
		t.Fatalf("tick rows %v", rows)
	}

	too := `["` + strings.Repeat(`x","`, MaxBalanceAddresses) + `y"]`
	for _, body := range []string{`{"addresses":[]}`, `{}`, `{"addresses":` + too + `}`} {
		if code, _ := drc20Post(t, handler, body); code != http.StatusBadRequest {
			t.Fatalf("body %.30s: code %d", body, code)
		}
	}
}
