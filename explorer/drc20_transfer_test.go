package explorer

import (
	"dogeuni-indexer/models"
	"testing"
)

const multiTransferActivation = 100

func newTransferExplorer(t *testing.T, balance int64) *Explorer {
	e := newTestExplorer(t, &models.Drc20Info{}, &models.Drc20Collect{}, &models.Drc20CollectAddress{}, &models.Drc20Revert{})
	e.multiTransferHeight = multiTransferActivation
	holder := &models.Drc20CollectAddress{Tick: "TICK", HolderAddress: "H", AmtSum: models.NewNumber(balance)}
	if err := e.dbc.DB.Create(holder).Error; err != nil {
		t.Fatal(err)
	}
	return e
}

func transferOrder(t *testing.T, e *Explorer, to string, amt, height int64) *models.Drc20Info {
	order := &models.Drc20Info{Op: "transfer", Tick: "TICK", Amt: models.NewNumber(amt), HolderAddress: "H",
		ToAddress: to, TxHash: "t1", BlockNumber: height, OrderStatus: 1}
	if err := e.dbc.DB.Create(order).Error; err != nil {
		t.Fatal(err)
	}
	return order
}

func balances(t *testing.T, e *Explorer) map[string]int64 {
	var rows []models.Drc20CollectAddress
	if err := e.dbc.DB.Where("tick = ?", "TICK").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	for _, r := range rows {
		got[r.HolderAddress] = r.AmtSum.Int().Int64()
	}
	return got
}

func wantBalances(t *testing.T, e *Explorer, want map[string]int64) {
	t.Helper()
	got := balances(t, e)
	if len(got) != len(want) {
		t.Fatalf("balances %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("balances %v, want %v", got, want)
		}
	}
}

func TestTransferCreditsEachRecipient(t *testing.T) {
	e := newTransferExplorer(t, 100)
	order := transferOrder(t, e, "A,B", 10, multiTransferActivation)
	if err := e.verify.VerifyDrc20(order); err != nil {
		t.Fatal(err)
	}
	if err := e.drc20Transfer(order); err != nil {
		t.Fatal(err)
	}
	wantBalances(t, e, map[string]int64{"H": 80, "A": 10, "B": 10})

	var count int64
	e.dbc.DB.Model(&models.Drc20Revert{}).Where("tx_hash = ?", "t1").Count(&count)
	if count != 2 {
		t.Fatalf("revert rows %d, want one per recipient", count)
	}
	stored := &models.Drc20Info{}
	if err := e.dbc.DB.Where("tx_hash = ?", "t1").First(stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.OrderStatus != 0 {
		t.Fatalf("order status %d", stored.OrderStatus)
	}
}

// the ledger below the activation height has to stay reproducible
func TestTransferKeepsJoinedHolderBeforeActivation(t *testing.T) {
	for _, activation := range []int64{multiTransferActivation, 0} {
		e := newTransferExplorer(t, 100)
		e.multiTransferHeight = activation
		if err := e.drc20Transfer(transferOrder(t, e, "A,B", 10, multiTransferActivation-1)); err != nil {
			t.Fatal(err)
		}
		wantBalances(t, e, map[string]int64{"H": 90, "A,B": 10})
	}
}

func TestTransferToOneRecipientIsUnchanged(t *testing.T) {
	e := newTransferExplorer(t, 100)
	if err := e.drc20Transfer(transferOrder(t, e, "A", 10, multiTransferActivation)); err != nil {
		t.Fatal(err)
	}
	wantBalances(t, e, map[string]int64{"H": 90, "A": 10})
}

// the balance covers the first recipient only: nobody is paid, not just the second one
func TestTransferIsAllOrNothing(t *testing.T) {
	e := newTransferExplorer(t, 15)
	if err := e.drc20Transfer(transferOrder(t, e, "A,B", 10, multiTransferActivation)); err == nil {
		t.Fatal("transfer above the balance passed")
	}
	wantBalances(t, e, map[string]int64{"H": 15})

	var count int64
	e.dbc.DB.Model(&models.Drc20Revert{}).Count(&count)
	if count != 0 {
		t.Fatalf("revert rows %d after a failed transfer", count)
	}
}

func TestTransferVerifyChargesEveryRecipient(t *testing.T) {
	e := newTransferExplorer(t, 15)
	order := &models.Drc20Info{Op: "transfer", Tick: "TICK", Amt: models.NewNumber(10), HolderAddress: "H", ToAddress: "A,B"}
	if err := e.verify.VerifyDrc20(order); err == nil {
		t.Fatal("verify passed a transfer above the balance")
	}
}

func TestTransferForkRestoresEveryRecipient(t *testing.T) {
	e := newTransferExplorer(t, 100)
	if err := e.drc20Transfer(transferOrder(t, e, "A,B", 10, multiTransferActivation)); err != nil {
		t.Fatal(err)
	}
	tx := e.dbc.DB.Begin()
	if err := e.drc20Fork(tx, multiTransferActivation-1); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	wantBalances(t, e, map[string]int64{"H": 100, "A": 0, "B": 0})
}
