package storage

import (
	"dogeuni-indexer/models"
	"errors"
	"fmt"
	"math/big"

	"github.com/dogecoinw/doged/btcutil"
	"github.com/dogecoinw/doged/chaincfg"
	"github.com/dogecoinw/doged/txscript"
	"gorm.io/gorm"
)

// prize-v1 state transitions (docs/prize-v1.md). Each function checks the rules that
// depend on state and applies the op inside tx; any error leaves the caller to roll
// tx back, so a failed op changes nothing. Rules on the inscription alone (address
// form, timeout range, entry count) are checked by the caller before.

const (
	PrizeTimeoutMin    = 1440
	PrizeTimeoutMax    = 525600
	PrizeMaxEntries    = 50
	prizeEntitlementPK = "pool_id = ? AND address = ?"
)

// PrizeReserve is the pool's reserve address, P2SH(OP_RETURN <"prize-v1:" + pool_id>).
// The redeem script fails as soon as it runs, so nobody can spend the reserve's
// outputs, and so nobody can send an inscription as the reserve.
func PrizeReserve(poolID string) (string, error) {
	script, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData([]byte("prize-v1:" + poolID)).Script()
	if err != nil {
		return "", err
	}
	addr, err := btcutil.NewAddressScriptHash(script, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}
	return addr.EncodeAddress(), nil
}

func prizePool(tx *gorm.DB, poolID string) (*models.PrizePool, error) {
	pool := &models.PrizePool{}
	err := tx.Where("pool_id = ?", poolID).First(pool).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("pool %s not found", poolID)
	}
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// prizePending returns the deposits that are neither refunded nor settled, given
// the pool has settled the deposits at heights <= upto.
func prizePending(tx *gorm.DB, poolID string, upto int64) ([]*models.PrizeDeposit, error) {
	var deposits []*models.PrizeDeposit
	err := tx.Where("pool_id = ? AND refunded = ? AND block_number > ?", poolID, false, upto).
		Order("block_number, id").Find(&deposits).Error
	return deposits, err
}

func prizeSum(deposits []*models.PrizeDeposit) *big.Int {
	sum := big.NewInt(0)
	for _, d := range deposits {
		sum.Add(sum, d.Amt.Int())
	}
	return sum
}

// PrizeClosed reports closed(h): status closed, or expired at h, which needs a
// pending deposit and h > the oldest pending deposit's height + timeout.
func PrizeClosed(tx *gorm.DB, pool *models.PrizePool, height int64) (bool, error) {
	if pool.Status == models.PrizeStatusClosed {
		return true, nil
	}
	pending, err := prizePending(tx, pool.PoolId, pool.Upto)
	if err != nil {
		return false, err
	}
	return len(pending) > 0 && height > pending[0].BlockNumber+pool.Timeout, nil
}

// PrizeBalance is the reserve's DRC-20 balance of tick; no row means 0.
func PrizeBalance(tx *gorm.DB, tick, reserve string) (*big.Int, error) {
	addr := &models.Drc20CollectAddress{}
	err := tx.Where("tick = ? AND holder_address = ?", tick, reserve).First(addr).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return big.NewInt(0), nil
	}
	if err != nil {
		return nil, err
	}
	return new(big.Int).Set(addr.AmtSum.Int()), nil
}

func prizeOpenPool(tx *gorm.DB, poolID string, height int64) (*models.PrizePool, error) {
	pool, err := prizePool(tx, poolID)
	if err != nil {
		return nil, err
	}
	closed, err := PrizeClosed(tx, pool, height)
	if err != nil {
		return nil, err
	}
	if closed {
		return nil, fmt.Errorf("pool %s is closed", poolID)
	}
	return pool, nil
}

func prizeClosedPool(tx *gorm.DB, poolID string, height int64) (*models.PrizePool, error) {
	pool, err := prizePool(tx, poolID)
	if err != nil {
		return nil, err
	}
	closed, err := PrizeClosed(tx, pool, height)
	if err != nil {
		return nil, err
	}
	if !closed {
		return nil, fmt.Errorf("pool %s is not closed", poolID)
	}
	return pool, nil
}

func savePrizePool(tx *gorm.DB, pool *models.PrizePool) error {
	return tx.Model(&models.PrizePool{}).Where("pool_id = ?", pool.PoolId).Updates(map[string]interface{}{
		"settler":            pool.Settler,
		"seq":                pool.Seq,
		"upto":               pool.Upto,
		"last_settle_height": pool.LastSettleHeight,
		"owed":               pool.Owed,
		"status":             pool.Status,
	}).Error
}

// newPrizePool is the state a create leaves; the fork replay starts from it too.
func newPrizePool(info *models.PrizeInfo, reserve string) *models.PrizePool {
	return &models.PrizePool{
		PoolId:           info.PoolId,
		Tick:             info.Tick,
		Creator:          info.HolderAddress,
		Settler:          info.Settler,
		Timeout:          info.Timeout,
		ReservesAddress:  reserve,
		Seq:              0,
		Upto:             info.BlockNumber - 1,
		LastSettleHeight: info.BlockNumber,
		Owed:             models.NewNumber(0),
		Status:           models.PrizeStatusOpen,
		BlockNumber:      info.BlockNumber,
	}
}

func (db *DBClient) PrizeCreate(tx *gorm.DB, info *models.PrizeInfo, reserve string) error {
	err := tx.Where("tick = ?", info.Tick).First(&models.Drc20Collect{}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("tick %s is not deployed", info.Tick)
	}
	if err != nil {
		return err
	}
	if info.Timeout < PrizeTimeoutMin || info.Timeout > PrizeTimeoutMax {
		return fmt.Errorf("timeout %d out of [%d, %d]", info.Timeout, PrizeTimeoutMin, PrizeTimeoutMax)
	}
	var n int64
	if err := tx.Model(&models.PrizePool{}).Where("pool_id = ?", info.PoolId).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("pool %s already exists", info.PoolId)
	}
	return tx.Create(newPrizePool(info, reserve)).Error
}

func (db *DBClient) PrizeDeposit(tx *gorm.DB, info *models.PrizeInfo) error {
	pool, err := prizeOpenPool(tx, info.PoolId, info.BlockNumber)
	if err != nil {
		return err
	}
	if info.Amt.Int().Sign() <= 0 {
		return fmt.Errorf("deposit amt must be > 0")
	}
	if err := db.TransferDrc20(tx, pool.Tick, info.HolderAddress, pool.ReservesAddress, info.Amt.Int(), info.TxHash, info.BlockNumber, false); err != nil {
		return err
	}
	return tx.Create(&models.PrizeDeposit{
		PoolId:        pool.PoolId,
		TxHash:        info.TxHash,
		HolderAddress: info.HolderAddress,
		Amt:           (*models.Number)(new(big.Int).Set(info.Amt.Int())),
		Memo:          info.Memo,
		BlockNumber:   info.BlockNumber,
	}).Error
}

func prizeEntitlement(tx *gorm.DB, poolID, address string) (*models.PrizeEntitlement, error) {
	ent := &models.PrizeEntitlement{}
	err := tx.Where(prizeEntitlementPK, poolID, address).First(ent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ent, nil
}

// PrizeSettle applies a settle whose entries the caller has checked for form
// (count, address, not the reserve, no duplicates).
func (db *DBClient) PrizeSettle(tx *gorm.DB, info *models.PrizeInfo, entries []models.PrizeEntry) error {
	pool, err := prizeOpenPool(tx, info.PoolId, info.BlockNumber)
	if err != nil {
		return err
	}
	if info.HolderAddress != pool.Settler {
		return fmt.Errorf("%s is not the settler", info.HolderAddress)
	}
	if info.Seq != pool.Seq+1 {
		return fmt.Errorf("seq %d, want %d", info.Seq, pool.Seq+1)
	}
	if info.Upto < pool.Upto || info.Upto >= info.BlockNumber {
		return fmt.Errorf("upto %d out of [%d, %d)", info.Upto, pool.Upto, info.BlockNumber)
	}

	owed := new(big.Int).Set(pool.Owed.Int())
	ents := make([]*models.PrizeEntitlement, len(entries))
	for i, entry := range entries {
		ent, err := prizeEntitlement(tx, pool.PoolId, entry.Address)
		if err != nil {
			return err
		}
		if ent == nil {
			ent = &models.PrizeEntitlement{PoolId: pool.PoolId, Address: entry.Address, Cum: models.NewNumber(0), Claimed: models.NewNumber(0)}
		}
		if entry.Cum.Int().Cmp(ent.Cum.Int()) < 0 {
			return fmt.Errorf("cum of %s decreases from %s to %s", entry.Address, ent.Cum, entry.Cum)
		}
		owed.Add(owed, new(big.Int).Sub(entry.Cum.Int(), ent.Cum.Int()))
		ent.Cum = (*models.Number)(new(big.Int).Set(entry.Cum.Int()))
		ents[i] = ent
	}

	pending, err := prizePending(tx, pool.PoolId, info.Upto)
	if err != nil {
		return err
	}
	balance, err := PrizeBalance(tx, pool.Tick, pool.ReservesAddress)
	if err != nil {
		return err
	}
	if need := new(big.Int).Add(owed, prizeSum(pending)); need.Cmp(balance) > 0 {
		return fmt.Errorf("owed %s + pending %s exceeds balance %s", owed, prizeSum(pending), balance)
	}

	for i, ent := range ents {
		if err := tx.Save(ent).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.PrizeSettleEntry{
			PoolId:      pool.PoolId,
			TxHash:      info.TxHash,
			Address:     entries[i].Address,
			Cum:         ent.Cum,
			BlockNumber: info.BlockNumber,
		}).Error; err != nil {
			return err
		}
	}

	pool.Owed = (*models.Number)(owed)
	pool.Seq = info.Seq
	pool.Upto = info.Upto
	pool.LastSettleHeight = info.BlockNumber
	return savePrizePool(tx, pool)
}

// PrizeClaim pays the holder cum - claimed and returns the amount.
func (db *DBClient) PrizeClaim(tx *gorm.DB, info *models.PrizeInfo) (*big.Int, error) {
	pool, err := prizePool(tx, info.PoolId)
	if err != nil {
		return nil, err
	}
	ent, err := prizeEntitlement(tx, pool.PoolId, info.HolderAddress)
	if err != nil {
		return nil, err
	}
	if ent == nil {
		return nil, fmt.Errorf("%s has nothing to claim", info.HolderAddress)
	}
	amt := new(big.Int).Sub(ent.Cum.Int(), ent.Claimed.Int())
	if amt.Sign() <= 0 {
		return nil, fmt.Errorf("%s has nothing to claim", info.HolderAddress)
	}
	if err := db.TransferDrc20(tx, pool.Tick, pool.ReservesAddress, info.HolderAddress, amt, info.TxHash, info.BlockNumber, false); err != nil {
		return nil, err
	}
	if err := tx.Model(&models.PrizeEntitlement{}).Where(prizeEntitlementPK, pool.PoolId, info.HolderAddress).
		Update("claimed", ent.Cum).Error; err != nil {
		return nil, err
	}
	pool.Owed = (*models.Number)(new(big.Int).Sub(pool.Owed.Int(), amt))
	return amt, savePrizePool(tx, pool)
}

// PrizeRefund pays the holder back its pending deposits, closes the pool and
// returns the amount.
func (db *DBClient) PrizeRefund(tx *gorm.DB, info *models.PrizeInfo) (*big.Int, error) {
	pool, err := prizeClosedPool(tx, info.PoolId, info.BlockNumber)
	if err != nil {
		return nil, err
	}
	var mine []*models.PrizeDeposit
	err = tx.Where("pool_id = ? AND holder_address = ? AND refunded = ? AND block_number > ?", pool.PoolId, info.HolderAddress, false, pool.Upto).
		Find(&mine).Error
	if err != nil {
		return nil, err
	}
	if len(mine) == 0 {
		return nil, fmt.Errorf("%s has no pending deposit", info.HolderAddress)
	}
	amt := prizeSum(mine)
	if err := db.TransferDrc20(tx, pool.Tick, pool.ReservesAddress, info.HolderAddress, amt, info.TxHash, info.BlockNumber, false); err != nil {
		return nil, err
	}
	for _, d := range mine {
		if err := tx.Model(&models.PrizeDeposit{}).Where("id = ?", d.ID).Update("refunded", true).Error; err != nil {
			return nil, err
		}
	}
	pool.Status = models.PrizeStatusClosed
	return amt, savePrizePool(tx, pool)
}

// PrizeWithdraw pays the creator balance - owed - pending of a closed pool and
// returns the amount.
func (db *DBClient) PrizeWithdraw(tx *gorm.DB, info *models.PrizeInfo) (*big.Int, error) {
	pool, err := prizeClosedPool(tx, info.PoolId, info.BlockNumber)
	if err != nil {
		return nil, err
	}
	if info.HolderAddress != pool.Creator {
		return nil, fmt.Errorf("%s is not the creator", info.HolderAddress)
	}
	pending, err := prizePending(tx, pool.PoolId, pool.Upto)
	if err != nil {
		return nil, err
	}
	balance, err := PrizeBalance(tx, pool.Tick, pool.ReservesAddress)
	if err != nil {
		return nil, err
	}
	amt := new(big.Int).Sub(balance, pool.Owed.Int())
	amt.Sub(amt, prizeSum(pending))
	if amt.Sign() <= 0 {
		return nil, fmt.Errorf("nothing to withdraw")
	}
	return amt, db.TransferDrc20(tx, pool.Tick, pool.ReservesAddress, info.HolderAddress, amt, info.TxHash, info.BlockNumber, false)
}

// PrizeConfig changes the settler to one the caller has checked for form.
func (db *DBClient) PrizeConfig(tx *gorm.DB, info *models.PrizeInfo) error {
	pool, err := prizeOpenPool(tx, info.PoolId, info.BlockNumber)
	if err != nil {
		return err
	}
	if info.HolderAddress != pool.Creator {
		return fmt.Errorf("%s is not the creator", info.HolderAddress)
	}
	pool.Settler = info.Settler
	return savePrizePool(tx, pool)
}

func (db *DBClient) PrizeClose(tx *gorm.DB, info *models.PrizeInfo) error {
	pool, err := prizeOpenPool(tx, info.PoolId, info.BlockNumber)
	if err != nil {
		return err
	}
	if info.HolderAddress != pool.Creator {
		return fmt.Errorf("%s is not the creator", info.HolderAddress)
	}
	pool.Status = models.PrizeStatusClosed
	return savePrizePool(tx, pool)
}

// PrizeRecompute rebuilds prize_pool, prize_entitlement and the refunded flags of
// prize_deposit for one pool by replaying its remaining successful ops in order. It
// runs on a fork after the rows above the fork height are deleted. Without a
// successful create the pool is removed entirely.
func (db *DBClient) PrizeRecompute(tx *gorm.DB, poolID string) error {
	var ops []*models.PrizeInfo
	err := tx.Where("pool_id = ? AND order_status = 0", poolID).Order("block_number, id").Find(&ops).Error
	if err != nil {
		return err
	}

	for _, m := range []interface{}{&models.PrizePool{}, &models.PrizeEntitlement{}} {
		if err := tx.Where("pool_id = ?", poolID).Delete(m).Error; err != nil {
			return err
		}
	}
	if len(ops) == 0 || ops[0].Op != "create" {
		for _, m := range []interface{}{&models.PrizeDeposit{}, &models.PrizeSettleEntry{}} {
			if err := tx.Where("pool_id = ?", poolID).Delete(m).Error; err != nil {
				return err
			}
		}
		return nil
	}

	reserve, err := PrizeReserve(poolID)
	if err != nil {
		return err
	}
	pool := newPrizePool(ops[0], reserve)

	var deposits []*models.PrizeDeposit
	if err := tx.Where("pool_id = ?", poolID).Find(&deposits).Error; err != nil {
		return err
	}
	byTx := make(map[string]*models.PrizeDeposit, len(deposits))
	for _, d := range deposits {
		d.Refunded = false
		byTx[d.TxHash] = d
	}

	ents := map[string]*models.PrizeEntitlement{}
	var order []string
	var seen []*models.PrizeDeposit
	for _, op := range ops[1:] {
		switch op.Op {
		case "deposit":
			d := byTx[op.TxHash]
			if d == nil {
				return fmt.Errorf("prize recompute: deposit %s has no prize_deposit row", op.TxHash)
			}
			seen = append(seen, d)
		case "settle":
			var entries []*models.PrizeSettleEntry
			if err := tx.Where("tx_hash = ?", op.TxHash).Order("id").Find(&entries).Error; err != nil {
				return err
			}
			for _, entry := range entries {
				ent := ents[entry.Address]
				if ent == nil {
					ent = &models.PrizeEntitlement{PoolId: poolID, Address: entry.Address, Claimed: models.NewNumber(0)}
					ents[entry.Address] = ent
					order = append(order, entry.Address)
				}
				ent.Cum = entry.Cum
			}
			pool.Seq = op.Seq
			pool.Upto = op.Upto
			pool.LastSettleHeight = op.BlockNumber
		case "claim":
			if ent := ents[op.HolderAddress]; ent != nil {
				ent.Claimed = ent.Cum
			}
		case "refund":
			for _, d := range seen {
				if d.HolderAddress == op.HolderAddress && !d.Refunded && d.BlockNumber > pool.Upto {
					d.Refunded = true
				}
			}
			pool.Status = models.PrizeStatusClosed
		case "config":
			pool.Settler = op.Settler
		case "close":
			pool.Status = models.PrizeStatusClosed
		}
	}

	owed := big.NewInt(0)
	for _, addr := range order {
		ent := ents[addr]
		owed.Add(owed, new(big.Int).Sub(ent.Cum.Int(), ent.Claimed.Int()))
		if err := tx.Create(ent).Error; err != nil {
			return err
		}
	}
	pool.Owed = (*models.Number)(owed)
	if err := tx.Create(pool).Error; err != nil {
		return err
	}
	for _, d := range deposits {
		if err := tx.Model(&models.PrizeDeposit{}).Where("id = ?", d.ID).Update("refunded", d.Refunded).Error; err != nil {
			return err
		}
	}
	return nil
}
