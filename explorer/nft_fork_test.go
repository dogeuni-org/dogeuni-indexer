package explorer

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func TestNftForkRoundTrip(t *testing.T) {
	dbc := storage.NewSqliteClient(utils.SqliteConfig{Switch: true, Database: filepath.Join(t.TempDir(), "t.db")})
	if err := dbc.DB.AutoMigrate(&models.NftInfo{}, &models.NftCollect{}, &models.NftCollectAddress{}, &models.NftRevert{}); err != nil {
		t.Fatal(err)
	}
	e := &Explorer{dbc: dbc}

	step := func(height int64, f func(tx *gorm.DB) error) {
		tx := dbc.DB.Begin()
		if err := f(tx); err != nil {
			tx.Rollback()
			t.Fatalf("height %d: %v", height, err)
		}
		tx.Commit()
	}

	step(100, func(tx *gorm.DB) error {
		return dbc.NftDeploy(tx, &models.NftInfo{Tick: "AINFT", Total: 10, HolderAddress: "A", TxHash: "d", BlockNumber: 100})
	})
	step(101, func(tx *gorm.DB) error {
		return dbc.NftMint(tx, &models.NftInfo{Tick: "AINFT", HolderAddress: "A", TxHash: "m1", BlockNumber: 101})
	})
	step(102, func(tx *gorm.DB) error {
		return dbc.NftMint(tx, &models.NftInfo{Tick: "AINFT", HolderAddress: "B", TxHash: "m2", BlockNumber: 102})
	})
	step(103, func(tx *gorm.DB) error {
		return dbc.NftTransfer(tx, &models.NftInfo{Tick: "AINFT", TickId: 1, HolderAddress: "A", ToAddress: "C", BlockNumber: 103})
	})
	// a transfer of a token the sender does not hold must fail
	bad := dbc.DB.Begin()
	if err := dbc.NftTransfer(bad, &models.NftInfo{Tick: "AINFT", TickId: 2, HolderAddress: "A", ToAddress: "C", BlockNumber: 103}); err == nil {
		t.Fatal("expected transfer of unheld token to fail")
	}
	bad.Rollback()

	collect := func() models.NftCollect {
		c := models.NftCollect{}
		dbc.DB.Where("tick = ?", "AINFT").First(&c)
		return c
	}
	holder := func(id int64) string {
		a := models.NftCollectAddress{}
		dbc.DB.Where("tick = ? AND tick_id = ?", "AINFT", id).First(&a)
		return a.HolderAddress
	}
	if c := collect(); c.TickSum != 2 || c.Transactions != 3 || holder(1) != "C" || holder(2) != "B" {
		t.Fatalf("after apply: sum=%d tx=%d #1=%s #2=%s", c.TickSum, c.Transactions, holder(1), holder(2))
	}

	fork := func(height int64) {
		step(height, func(tx *gorm.DB) error {
			if err := e.nftFork(tx, height); err != nil {
				return err
			}
			return tx.Where("block_number > ?", height).Delete(&models.NftRevert{}).Error
		})
	}

	fork(102) // undo the transfer
	if c := collect(); c.TickSum != 2 || c.Transactions != 2 || holder(1) != "A" {
		t.Fatalf("after fork 102: sum=%d tx=%d #1=%s", c.TickSum, c.Transactions, holder(1))
	}

	fork(101) // undo the second mint
	if c := collect(); c.TickSum != 1 || c.Transactions != 1 || holder(2) != "" {
		t.Fatalf("after fork 101: sum=%d tx=%d #2=%s", c.TickSum, c.Transactions, holder(2))
	}

	fork(99) // undo everything including deploy
	var n1, n2, n3 int64
	dbc.DB.Model(&models.NftCollect{}).Count(&n1)
	dbc.DB.Model(&models.NftCollectAddress{}).Count(&n2)
	dbc.DB.Model(&models.NftRevert{}).Count(&n3)
	if n1+n2+n3 != 0 {
		t.Fatalf("after fork 99: collect=%d address=%d revert=%d", n1, n2, n3)
	}
}
