package explorer

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/dogecoinw/doged/btcjson"
	"gorm.io/gorm"
)

func newTestExplorer(t *testing.T, tables ...interface{}) *Explorer {
	return newTestExplorerAt(t, filepath.Join(t.TempDir(), "t.db"), tables...)
}

func newTestExplorerAt(t *testing.T, dsn string, tables ...interface{}) *Explorer {
	base := storage.NewSqliteClient(utils.SqliteConfig{Switch: true, Database: dsn})
	if err := base.DB.AutoMigrate(tables...); err != nil {
		t.Fatal(err)
	}
	dbc, fault := base.WithFaultRecorder()
	return &Explorer{dbc: dbc, fault: fault, verify: NewVerifys(dbc)}
}

// fork() drops drc20_info before any protocol fork runs; a deploy above the fork
// height must still take its tick with it
func TestDrc20DeployFork(t *testing.T) {
	e := newTestExplorer(t, &models.Drc20Info{}, &models.Drc20Collect{}, &models.Drc20CollectAddress{}, &models.Drc20Revert{})
	db := e.dbc.DB

	deploy := func(tick, hash string, height int64) {
		info := &models.Drc20Info{Op: "deploy", Tick: tick, Max: models.NewNumber(1e6), Lim: models.NewNumber(1e3), HolderAddress: "A", TxHash: hash, BlockNumber: height}
		if err := db.Create(info).Error; err != nil {
			t.Fatal(err)
		}
		if err := e.drc20Deploy(info); err != nil {
			t.Fatal(err)
		}
	}
	apply := func(f func(tx *gorm.DB) error) {
		tx := db.Begin()
		if err := f(tx); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		tx.Commit()
	}

	deploy("KEEP", "d-keep", 90)
	deploy("GONE", "d-gone", 100)
	apply(func(tx *gorm.DB) error { return e.dbc.MintDrc20(tx, "KEEP", "A", big.NewInt(500), "m0", 101, false) })
	apply(func(tx *gorm.DB) error { return e.dbc.MintDrc20(tx, "GONE", "A", big.NewInt(1000), "m1", 101, false) })
	apply(func(tx *gorm.DB) error {
		return e.dbc.TransferDrc20(tx, "GONE", "A", "B", big.NewInt(400), "t1", 102, false)
	})

	// same sequence as fork(): capture deploys, drop infos, revert balances, drop ticks
	height := int64(99)
	apply(func(tx *gorm.DB) error {
		var deploys []string
		if err := tx.Model(&models.Drc20Info{}).Where("op = ? AND block_number > ?", "deploy", height).Pluck("tx_hash", &deploys).Error; err != nil {
			return err
		}
		if err := tx.Where("block_number > ?", height).Delete(&models.Drc20Info{}).Error; err != nil {
			return err
		}
		if err := e.drc20Fork(tx, height); err != nil {
			return err
		}
		if err := e.drc20DeployFork(tx, deploys); err != nil {
			return err
		}
		return tx.Where("block_number > ?", height).Delete(&models.Drc20Revert{}).Error
	})

	var gone, gonea, keep int64
	db.Model(&models.Drc20Collect{}).Where("tick = ?", "GONE").Count(&gone)
	db.Model(&models.Drc20CollectAddress{}).Where("tick = ?", "GONE").Count(&gonea)
	db.Model(&models.Drc20Collect{}).Where("tick = ?", "KEEP").Count(&keep)
	if gone != 0 || gonea != 0 || keep != 1 {
		t.Fatalf("GONE collect=%d address=%d, KEEP collect=%d", gone, gonea, keep)
	}

	keepA := &models.Drc20CollectAddress{}
	db.Where("tick = ? AND holder_address = ?", "KEEP", "A").First(keepA)
	if keepA.AmtSum.Int().Sign() != 0 {
		t.Fatalf("KEEP balance of A after fork = %s, want 0", keepA.AmtSum.String())
	}

	// with the tick gone, a redeploy on the new branch is accepted again
	deploy("GONE", "d-gone-2", 100)
}

// an output without an address used to panic the scan loop; it now invalidates the inscription
func TestDrc20DecodeOutputWithoutAddress(t *testing.T) {
	e := newTestExplorer(t, &models.Drc20Info{})
	tx := &btcjson.TxRawResult{
		Hash: "h", Txid: "h",
		Vin:  []btcjson.Vin{{Txid: "prev"}},
		Vout: []btcjson.Vout{{Value: 0.001, ScriptPubKey: btcjson.ScriptPubKeyResult{Type: "nulldata"}}},
	}
	_, err := e.drc20Decode(tx, []byte(`{"p":"drc-20","op":"deploy","tick":"TEST","max":"100","lim":"1"}`), 1)
	if err == nil {
		t.Fatal("expected decode error for output without address")
	}

	if _, err := outputAddress(tx, 1); err == nil {
		t.Fatal("expected error for missing output index")
	}
}
