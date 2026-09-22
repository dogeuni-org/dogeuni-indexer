package explorer

import (
	"dogeuni-indexer/utils"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dogecoinw/doged/btcjson"
)

const activation = 100

func nodeTx(t *testing.T, sats ...int64) *btcjson.TxRawResult {
	tx := &btcjson.TxRawResult{}
	for _, s := range sats {
		var v float64
		if err := json.Unmarshal([]byte(fmt.Sprintf("%d.%08d", s/utils.SatoshisPerDoge, s%utils.SatoshisPerDoge)), &v); err != nil {
			t.Fatal(err)
		}
		tx.Vout = append(tx.Vout, btcjson.Vout{Value: v})
	}
	return tx
}

func TestMintRepeat(t *testing.T) {
	e := &Explorer{amountHeight: activation}
	var legacyRejected []int64
	for r := int64(1); r <= MintMaxRepeat; r++ {
		v := nodeTx(t, r*MintUnitSats).Vout[0].Value
		if repeat, ok := e.mintRepeat(v, activation); !ok || repeat != r {
			t.Fatalf("exact: %d units read as %d ok=%v", r, repeat, ok)
		}
		if _, ok := e.mintRepeat(v, activation-1); !ok {
			legacyRejected = append(legacyRejected, r)
		}
	}
	// blocks before activation must keep the legacy result
	if fmt.Sprint(legacyRejected) != "[9 13 18 26]" {
		t.Fatalf("legacy rejects %v, want [9 13 18 26]", legacyRejected)
	}
	for _, sats := range []int64{MintUnitSats + 1, MintUnitSats - 1, 31 * MintUnitSats} {
		v := nodeTx(t, sats).Vout[0].Value
		if _, ok := e.mintRepeat(v, activation); ok {
			t.Fatalf("exact: %d sats accepted as a mint", sats)
		}
	}
}

func TestVoutSats(t *testing.T) {
	// 7 sats is one of the values the legacy conversion reads as 8
	tx := nodeTx(t, 7, 50000000)
	for _, c := range []struct {
		amountHeight, height, want int64
	}{
		{activation, activation, 7},
		{activation, activation + 1, 7},
		{activation, activation - 1, 8},
		{0, activation, 8}, // not configured: legacy
	} {
		e := &Explorer{amountHeight: c.amountHeight}
		if got := e.voutSats(tx, 0, c.height).Int64(); got != c.want {
			t.Fatalf("amount_height=%d height=%d: got %d, want %d", c.amountHeight, c.height, got, c.want)
		}
	}
	e := &Explorer{amountHeight: activation}
	if got := e.voutSats(tx, 1, activation).Int64(); got != 50000000 {
		t.Fatalf("0.5 DOGE read as %d sats", got)
	}
}
