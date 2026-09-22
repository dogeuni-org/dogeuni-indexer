package explorer

import (
	"dogeuni-indexer/models"
	"testing"
)

var swapAddTables = []interface{}{&models.SwapLiquidity{}, &models.SwapV2Liquidity{}, &models.Drc20CollectAddress{}}

// Reserve states an add must handle: drained on one side (as 11 mainnet pools are),
// empty, unset, and healthy.
var addPools = []struct {
	name       string
	amt0, amt1 *models.Number
	ok         bool
}{
	{"drained0", models.NewNumber(0), models.NewNumber(500), false},
	{"drained1", models.NewNumber(500), models.NewNumber(0), false},
	{"empty", models.NewNumber(0), models.NewNumber(0), false},
	{"unset", nil, nil, false},
	{"healthy", models.NewNumber(1000), models.NewNumber(2000), true},
}

func fundHolder(t *testing.T, e *Explorer, ticks ...string) {
	for _, tick := range ticks {
		if err := e.dbc.DB.Create(&models.Drc20CollectAddress{Tick: tick, HolderAddress: "H", AmtSum: models.NewNumber(1e9)}).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestSwapAddReserves(t *testing.T) {
	for _, p := range addPools {
		t.Run(p.name, func(t *testing.T) {
			e := newTestExplorer(t, swapAddTables...)
			fundHolder(t, e, "AAAA", "BBBB")
			pool := &models.SwapLiquidity{Tick: "AAAA-SWAP-BBBB", Tick0: "AAAA", Tick1: "BBBB", Amt0: p.amt0, Amt1: p.amt1, LiquidityTotal: models.NewNumber(1000)}
			if err := e.dbc.DB.Create(pool).Error; err != nil {
				t.Fatal(err)
			}
			swap := &models.SwapInfo{Op: "add", Tick0: "AAAA", Tick1: "BBBB", Amt0: models.NewNumber(10), Amt1: models.NewNumber(20),
				Amt0Min: models.NewNumber(1), Amt1Min: models.NewNumber(1), HolderAddress: "H"}
			if err := e.verify.verifySwapAdd(e.dbc.DB, swap); (err == nil) != p.ok {
				t.Fatalf("pool %s/%s: got %v, want ok=%v", p.amt0, p.amt1, err, p.ok)
			}
		})
	}
}

func TestSwapV2AddReserves(t *testing.T) {
	for _, p := range addPools {
		t.Run(p.name, func(t *testing.T) {
			e := newTestExplorer(t, swapAddTables...)
			fundHolder(t, e, "AAAA", "BBBB")
			pool := &models.SwapV2Liquidity{PairId: "P", Tick0Id: "AAAA", Tick1Id: "BBBB", Tick0: "AAAA", Tick1: "BBBB", Amt0: p.amt0, Amt1: p.amt1, LiquidityTotal: models.NewNumber(1000)}
			if err := e.dbc.DB.Create(pool).Error; err != nil {
				t.Fatal(err)
			}
			swap := &models.SwapV2Info{Op: "add", PairId: "P", Amt0: models.NewNumber(10), Amt1: models.NewNumber(20),
				Amt0Min: models.NewNumber(1), Amt1Min: models.NewNumber(1), HolderAddress: "H"}
			if err := e.verify.verifySwapV2Add(e.dbc.DB, swap); (err == nil) != p.ok {
				t.Fatalf("pool %s/%s: got %v, want ok=%v", p.amt0, p.amt1, err, p.ok)
			}
		})
	}
}
