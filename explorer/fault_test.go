package explorer

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"dogeuni-indexer/models"
	"errors"
	"math/big"
	"path/filepath"
	"testing"

	"gorm.io/gorm"
)

var drc20Tables = []interface{}{&models.Drc20Info{}, &models.Drc20Collect{}, &models.Drc20CollectAddress{}, &models.Drc20Revert{}}

// injectQueryFault makes every query on e's database fail with a lost connection
// while *on is true.
func injectQueryFault(t *testing.T, e *Explorer, on *bool) {
	err := e.dbc.DB.Callback().Query().Before("gorm:query").Register("test:inject_fault", func(db *gorm.DB) {
		if *on {
			db.AddError(driver.ErrBadConn)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func deployTick(t *testing.T, e *Explorer, tick string) {
	info := &models.Drc20Info{Op: "deploy", Tick: tick, Max: models.NewNumber(1e12), Lim: models.NewNumber(1e12), HolderAddress: "A", TxHash: "deploy-" + tick, BlockNumber: 1}
	if err := e.dbc.DB.Create(info).Error; err != nil {
		t.Fatal(err)
	}
	if err := e.drc20Deploy(info); err != nil {
		t.Fatal(err)
	}
}

func pendingMint(t *testing.T, e *Explorer, tick, hash string) *models.Drc20Info {
	info := &models.Drc20Info{Op: "mint", Tick: tick, Amt: models.NewNumber(1), Repeat: 1, HolderAddress: "B", TxHash: hash, BlockNumber: 2, OrderStatus: 1}
	if err := e.dbc.DB.Create(info).Error; err != nil {
		t.Fatal(err)
	}
	return info
}

func mintRow(t *testing.T, e *Explorer, hash string) *models.Drc20Info {
	row := &models.Drc20Info{}
	if err := e.dbc.DB.Where("tx_hash = ?", hash).First(row).Error; err != nil {
		t.Fatal(err)
	}
	return row
}

// A database write that fails because the database is locked must leave the tx
// pending and abort the block, even though executeDrc20 flattens the cause into a
// string. A tx that is really invalid is still rejected with err_info.
func TestExecuteFaultAbortsBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	e := newTestExplorerAt(t, path+"?_busy_timeout=50", drc20Tables...)
	deployTick(t, e, "TEST")
	info := pendingMint(t, e, "TEST", "m1")

	// another writer holds the database
	other, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	ctx := context.Background()
	conn, err := other.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}

	execErr := e.executeDrc20(info)
	if execErr == nil {
		t.Fatal("mint succeeded while the database was locked")
	}
	abort := e.failTx(&models.Drc20Info{}, info.TxHash, execErr)
	if abort == nil || !errors.Is(abort, STORAGE_ERR) {
		t.Fatalf("want block abort on locked database, got %v (execute: %v)", abort, execErr)
	}

	if _, err := conn.ExecContext(ctx, "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	if row := mintRow(t, e, "m1"); row.ErrInfo != "" || row.OrderStatus != 1 {
		t.Fatalf("faulted tx must stay pending, got order_status=%d err_info=%q", row.OrderStatus, row.ErrInfo)
	}

	// rescan: executes now that the database is back
	if err := e.executeDrc20(mintRow(t, e, "m1")); err != nil {
		t.Fatal(err)
	}
	if row := mintRow(t, e, "m1"); row.OrderStatus != 0 {
		t.Fatalf("mint not applied on rescan: order_status=%d", row.OrderStatus)
	}

	// control: an invalid mint (unknown tick) is rejected, not retried
	bad := pendingMint(t, e, "NONE", "m2")
	execErr = e.executeDrc20(bad)
	if execErr == nil {
		t.Fatal("mint of unknown tick succeeded")
	}
	if abort := e.failTx(&models.Drc20Info{}, bad.TxHash, execErr); abort != nil {
		t.Fatalf("invalid tx must not abort the block: %v", abort)
	}
	if row := mintRow(t, e, "m2"); row.ErrInfo == "" {
		t.Fatal("invalid tx has no err_info")
	}
}

// A query fault inside Verify*, whose errors become "does not exist"-style strings,
// is still seen as a fault.
func TestVerifyQueryFaultIsRecorded(t *testing.T) {
	e := newTestExplorer(t, drc20Tables...)
	deployTick(t, e, "TEST")
	info := pendingMint(t, e, "TEST", "m1")

	on := true
	injectQueryFault(t, e, &on)
	execErr := e.executeDrc20(info)
	on = false
	if execErr == nil {
		t.Fatal("want execute error under query fault")
	}
	if abort := e.failTx(&models.Drc20Info{}, info.TxHash, execErr); !errors.Is(abort, STORAGE_ERR) || !errors.Is(abort, driver.ErrBadConn) {
		t.Fatalf("want abort wrapping the fault, got %v", abort)
	}
}

// Rows of a pass cut off before execute committed are decoded again; finished rows,
// valid or rejected, are not.
func TestDropPending(t *testing.T) {
	e := newTestExplorer(t, &models.Drc20Info{})
	rows := []*models.Drc20Info{
		{TxHash: "pending", OrderStatus: 1},
		{TxHash: "done", OrderStatus: 0},
		{TxHash: "rejected", OrderStatus: 1, ErrInfo: "invalid"},
	}
	for _, r := range rows {
		if err := e.dbc.DB.Create(r).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range rows {
		if err := e.dropPending(&models.Drc20Info{}, r.TxHash); err != nil {
			t.Fatal(err)
		}
	}
	var left []string
	e.dbc.DB.Model(&models.Drc20Info{}).Order("tx_hash").Pluck("tx_hash", &left)
	if len(left) != 2 || left[0] != "done" || left[1] != "rejected" {
		t.Fatalf("want [done rejected] kept, got %v", left)
	}
}

// The wdoge deposit that funds a swap commits on its own; executing the swap again
// after a fault must not credit it twice.
func TestDepositSwapOnce(t *testing.T) {
	e := newTestExplorer(t, append(drc20Tables, &models.WDogeInfo{})...)
	const tick = "WDOGE(WRAPPED-DOGE)"
	deployTick(t, e, tick)

	deposit := func() {
		tx := e.dbc.DB.Begin()
		w := &models.WDogeInfo{Op: "deposit-swap", Tick: tick, Amt: models.NewNumber(5), HolderAddress: "B", TxHash: "s1", BlockNumber: 2}
		if err := e.wdogeDepositSwap(tx, w); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		if err := tx.Commit().Error; err != nil {
			t.Fatal(err)
		}
	}
	deposit()
	deposit()

	bal := &models.Drc20CollectAddress{}
	if err := e.dbc.DB.Where("tick = ? AND holder_address = ?", tick, "B").First(bal).Error; err != nil {
		t.Fatal(err)
	}
	if bal.AmtSum.Int().Cmp(big.NewInt(5)) != 0 {
		t.Fatalf("deposit credited %s, want 5", bal.AmtSum.String())
	}
}
