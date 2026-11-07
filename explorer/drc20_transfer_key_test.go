package explorer

import (
	"dogeuni-indexer/models"
	"strings"
	"testing"

	"github.com/dogecoinw/doged/btcjson"
)

// secp256k1 generator, a key that is valid on the curve
const testPubkey = "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"

func TestP2pkhPubkey(t *testing.T) {
	sig := "47" + strings.Repeat("ab", 71)
	cases := []struct {
		name string
		in   *btcjson.ScriptSig
		want string
	}{
		{"p2pkh", &btcjson.ScriptSig{Hex: sig + "21" + testPubkey}, testPubkey},
		// input 0 of a commit tx on mainnet, spending from D5nZoAXHeTWDJchBHhemMhcKrjHSFNtVM8
		{"mainnet", &btcjson.ScriptSig{Hex: "483045022100e00b830015db07f47d5d9dd85daa533e8742df7ca683e20f1e9786d434a20d34022014920f157e12154040366c19c01739a881388b440acac355d03c12a5a8475c3f0121021cbee0f539b7506e66d52acaa4513a1228f765960ca8de2f1acf9678dea8852e"},
			"021cbee0f539b7506e66d52acaa4513a1228f765960ca8de2f1acf9678dea8852e"},
		{"no scriptSig", nil, ""},
		{"empty", &btcjson.ScriptSig{}, ""},
		{"p2sh with extra pushes", &btcjson.ScriptSig{Hex: "00" + sig + "21" + testPubkey}, ""},
		{"trailing push", &btcjson.ScriptSig{Hex: sig + "21" + testPubkey + "00"}, ""},
		{"uncompressed", &btcjson.ScriptSig{Hex: sig + "41" + "04" + strings.Repeat("ab", 64)}, ""},
		{"not on the curve", &btcjson.ScriptSig{Hex: sig + "21" + "02" + strings.Repeat("ff", 32)}, ""},
		{"not hex", &btcjson.ScriptSig{Hex: "zz"}, ""},
	}
	for _, c := range cases {
		if got := p2pkhPubkey(&btcjson.Vin{ScriptSig: c.in}); got != c.want {
			t.Fatalf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSaveTransferKey(t *testing.T) {
	e := newTestExplorer(t, &models.BoxCollectAddress{}, &models.BoxInfo{}, &models.CrossInfo{}, &models.Drc20Info{}, &models.Drc20TransferKey{}, &models.ExchangeInfo{}, &models.FileExchangeInfo{}, &models.FileInfo{}, &models.InviteInfo{}, &models.Meme20Info{}, &models.NftInfo{}, &models.PumpInfo{}, &models.StakeInfo{}, &models.StakeRewardInfo{}, &models.StakeV2Info{}, &models.SwapInfo{}, &models.SwapV2Info{}, &models.WDogeInfo{})
	in := &btcjson.Vin{Txid: "prev", Vout: 3, ScriptSig: &btcjson.ScriptSig{Hex: "01ab21" + testPubkey}}
	card := &models.Drc20Info{TxHash: "t1", BlockNumber: 100}

	// the second pass is a pending row decoded again after a restart
	for i := 0; i < 2; i++ {
		if err := e.saveTransferKey(card, in); err != nil {
			t.Fatal(err)
		}
	}
	// a transfer signed by a script records nothing and still succeeds
	if err := e.saveTransferKey(&models.Drc20Info{TxHash: "t2", BlockNumber: 100}, &btcjson.Vin{}); err != nil {
		t.Fatal(err)
	}
	var keys []models.Drc20TransferKey
	if err := e.dbc.DB.Find(&keys).Error; err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].SenderPubkey != testPubkey || keys[0].SenderTxid != "prev" || keys[0].SenderVout != 3 {
		t.Fatalf("keys %+v", keys)
	}

	tx := e.dbc.DB.Begin()
	if err := e.delInfo(tx, 99); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	var left int64
	e.dbc.DB.Model(&models.Drc20TransferKey{}).Count(&left)
	if left != 0 {
		t.Fatalf("%d keys survived the fork", left)
	}
}
