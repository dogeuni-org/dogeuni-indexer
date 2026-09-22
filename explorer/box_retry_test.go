package explorer

import (
	"dogeuni-indexer/models"
	"testing"
)

// a block re-scanned after a node/storage error runs its scheduled box finish again;
// a finish already recorded at that height must be skipped
func TestBoxFinishSkippedOnRescan(t *testing.T) {
	e := newTestExplorer(t, &models.BoxCollect{}, &models.BoxRevert{})
	db := e.dbc.DB
	box := &models.BoxCollect{Tick0: "BOXA", Tick1: "WDOGE(WRAPPED-DOGE)", LiqBlock: 200, LiqAmtFinish: models.NewNumber(5)}
	if err := db.Create(box).Error; err != nil {
		t.Fatal(err)
	}

	// no finish recorded yet: BoxFinish runs (and fails here, its swap tables are not migrated)
	if err := e.dbc.BoxDeployScheduled(db, 200); err == nil {
		t.Fatal("expected BoxFinish to run on first scan")
	}

	// finish recorded (pair stored sorted): the re-scan skips it
	if err := db.Create(&models.BoxRevert{Op: "finish", Tick0: "BOXA", Tick1: "WDOGE(WRAPPED-DOGE)", BlockNumber: 200}).Error; err != nil {
		t.Fatal(err)
	}
	if err := e.dbc.BoxDeployScheduled(db, 200); err != nil {
		t.Fatalf("re-scan should skip the finished box: %v", err)
	}

	// another box finishing at the same height with BOXA as its quote token does not count
	other := &models.BoxCollect{Tick0: "BOXB", Tick1: "BOXA", LiqBlock: 201, LiqAmtFinish: models.NewNumber(5)}
	db.Create(other)
	db.Create(&models.BoxRevert{Op: "finish", Tick0: "BOXA", Tick1: "WDOGE(WRAPPED-DOGE)", BlockNumber: 201})
	if err := e.dbc.BoxDeployScheduled(db, 201); err == nil {
		t.Fatal("finish of BOXB must not be skipped by an unrelated pair")
	}
}
