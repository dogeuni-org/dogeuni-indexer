package explorer

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"strings"
	"testing"
)

var pumpTradeTables = []interface{}{&models.PumpInfo{}, &models.PumpLiquidity{}, &models.PumpRevert{}, &models.InviteCollect{},
	&models.Drc20Collect{}, &models.Drc20CollectAddress{}, &models.Drc20Revert{}}

// Pool 1,000,000 WDOGE : 1,000,000 MEME. Buying with 10,000 WDOGE: 1% fee in, 9,900 in,
// 1,000,000*9,900/1,009,900 = 9,802 out, 1% fee out, 9,704 to the holder.
const pumpTradeOut = 9704

func runPumpTrade(t *testing.T, height int64, amt1Min int64) (*models.PumpInfo, error) {
	e := newTestExplorer(t, pumpTradeTables...)
	db := e.dbc.DB
	pool := &models.PumpLiquidity{Tick0Id: "WDOGE(WRAPPED-DOGE)", Tick1Id: "MEME", Amt0: models.NewNumber(1e6), Amt1: models.NewNumber(1e6), ReservesAddress: "R"}
	if err := db.Create(pool).Error; err != nil {
		t.Fatal(err)
	}
	for _, c := range []*models.Drc20CollectAddress{
		{Tick: "WDOGE(WRAPPED-DOGE)", HolderAddress: "H", AmtSum: models.NewNumber(1e6)},
		{Tick: "MEME", HolderAddress: "R", AmtSum: models.NewNumber(1e6)},
	} {
		if err := db.Create(c).Error; err != nil {
			t.Fatal(err)
		}
	}
	pump := &models.PumpInfo{Op: "trade", PairId: "WDOGE(WRAPPED-DOGE)", Tick0Id: "WDOGE(WRAPPED-DOGE)", Tick1Id: "MEME",
		Amt0: models.NewNumber(10000), Amt1Min: models.NewNumber(amt1Min), HolderAddress: "H", TxHash: "T", BlockNumber: height}
	if err := db.Create(pump).Error; err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	err := e.pumpTrade(tx, pump)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return pump, err
}

func TestPumpTradeMinOut(t *testing.T) {
	defer func(h int64) { storage.PumpMinOutHeight = h }(storage.PumpMinOutHeight)
	storage.PumpMinOutHeight = 100

	cases := []struct {
		name    string
		height  int64
		amt1Min int64
		ok      bool
	}{
		{"before activation, minimum ignored", 99, pumpTradeOut + 1, true},
		{"minimum met exactly", 100, pumpTradeOut, true},
		{"minimum zero", 100, 0, true},
		{"minimum above output", 100, pumpTradeOut + 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pump, err := runPumpTrade(t, c.height, c.amt1Min)
			if c.ok {
				if err != nil {
					t.Fatalf("trade failed: %v", err)
				}
				if got := pump.Amt1Out.Int().Int64(); got != pumpTradeOut {
					t.Fatalf("amt1_out = %d, want %d", got, pumpTradeOut)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "minimum output less than the limit") {
				t.Fatalf("trade below its minimum: err = %v, want the minimum output error", err)
			}
		})
	}
}
