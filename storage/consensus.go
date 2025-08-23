package storage

import (
	"fmt"
	"math"
	"math/big"

	"dogeuni-indexer/models"

	"gorm.io/gorm"
)

// ConsensusStake handles consensus stake operations
func (e *DBClient) ConsensusStake(tx *gorm.DB, consensus *models.ConsensusInfo, reservesAddress string) error {
	// Each independent stake: insert ConsensusStakeRecord, active, no stacking
	record := &models.ConsensusStakeRecord{
		StakeId:         consensus.StakeId, // Set by explorer to current txhash when staking
		HolderAddress:   consensus.HolderAddress,
		ReservesAddress: reservesAddress,
		Amt:             consensus.Amt,
		StakeBlock:      consensus.BlockNumber,
		Status:          "active",
		Score:           nil, // Score is nil when staking, calculated when unstaking
	}

	if err := tx.Create(record).Error; err != nil {
		return fmt.Errorf("create consensus stake record error: %v", err)
	}

	// Ledger migration: holder -> reserves
	if err := e.TransferDrc20(tx, "CARDI", consensus.HolderAddress, reservesAddress, consensus.Amt.Int(), consensus.TxHash, consensus.BlockNumber, false); err != nil {
		return fmt.Errorf("transfer doge to reserves address error: %v", err)
	}

	// Write rollback log: for fork rollback to revoke stake (fund migration + delete record)
	revert := &models.ConsensusRevert{
		FromAddress: consensus.HolderAddress,
		ToAddress:   reservesAddress,
		Op:          "stake",
		Amt:         consensus.Amt,
		// For stake, TxHash is also stakeId
		TxHash:      consensus.StakeId,
		BlockNumber: consensus.BlockNumber,
	}
	if err := tx.Create(revert).Error; err != nil {
		return fmt.Errorf("create consensus revert(stake) error: %v", err)
	}

	return nil
}

// ConsensusUnstake handles consensus unstake operations
func (e *DBClient) ConsensusUnstake(tx *gorm.DB, consensus *models.ConsensusInfo, reservesAddress string) error {
	// Unstake must provide stake_id, amount must match original stake, only full unstake allowed
	if consensus.StakeId == "" {
		return fmt.Errorf("unstake requires stake_id")
	}

	record := &models.ConsensusStakeRecord{}
	if err := tx.Where("stake_id = ? AND status = ?", consensus.StakeId, "active").First(record).Error; err != nil {
		return fmt.Errorf("query consensus stake record error: %v", err)
	}

	if record.HolderAddress != consensus.HolderAddress {
		return fmt.Errorf("holder address mismatch")
	}

	// Calculate score (independent method reusable by router)
	scoreVal := e.CalculateConsensusScore(record.Amt, record.StakeBlock, consensus.BlockNumber)

	// Write back record: close and record unstake_block and score
	unstakeBlock := consensus.BlockNumber

	if err := tx.Model(record).Updates(map[string]interface{}{
		"unstake_block": unstakeBlock,
		"status":        "closed",
		"score":         scoreVal.String(),
	}).Error; err != nil {
		return fmt.Errorf("update stake record error: %v", err)
	}

	// Ledger migration: reserves -> holder (full amount)
	if err := e.TransferDrc20(tx, "CARDI", reservesAddress, consensus.HolderAddress, record.Amt.Int(), consensus.TxHash, consensus.BlockNumber, false); err != nil {
		return fmt.Errorf("transfer doge from reserves address error: %v", err)
	}

	consensus.Amt = record.Amt

	// Write rollback log: for fork rollback to revoke unstake (fund migration + record reopen), amount recorded as full
	revert := &models.ConsensusRevert{
		FromAddress: reservesAddress,
		ToAddress:   consensus.HolderAddress,
		Op:          "unstake",
		Amt:         record.Amt,
		// For unstake, TxHash stores stakeId for record location
		TxHash:      consensus.StakeId,
		BlockNumber: consensus.BlockNumber,
	}
	if err := tx.Create(revert).Error; err != nil {
		return fmt.Errorf("create consensus revert(unstake) error: %v", err)
	}

	return nil
}

// ConsensusRevertStake revokes one stake (for fork rollback)
func (e *DBClient) ConsensusRevertStake(tx *gorm.DB, holderAddress, reservesAddress string, amt *big.Int, stakeId string, blockNumber int64) error {
	// Fund migration: reserves -> holder
	//if err := e.TransferDrc20(tx, "CARDI", reservesAddress, holderAddress, amt, "", blockNumber, true); err != nil {
	//	return fmt.Errorf("revert stake: transfer back error: %v", err)
	//}

	// Delete this independent stake record
	if err := tx.Where("stake_id = ?", stakeId).Delete(&models.ConsensusStakeRecord{}).Error; err != nil {
		return fmt.Errorf("revert stake: delete record error: %v", err)
	}
	return nil
}

// ConsensusRevertUnstake revokes one unstake (for fork rollback)
func (e *DBClient) ConsensusRevertUnstake(tx *gorm.DB, holderAddress, reservesAddress string, amt *big.Int, stakeId string, blockNumber int64) error {
	// Fund migration: holder -> reserves
	//if err := e.TransferDrc20(tx, "CARDI", holderAddress, reservesAddress, amt, "", blockNumber, true); err != nil {
	//	return fmt.Errorf("revert unstake: transfer back error: %v", err)
	//}

	// Reopen record: status=active, clear unstake info and score
	rec := &models.ConsensusStakeRecord{}
	if err := tx.Where("stake_id = ?", stakeId).First(rec).Error; err != nil {
		return fmt.Errorf("revert unstake: find record error: %v", err)
	}
	rec.Status = "active"
	rec.UnstakeBlock = nil
	rec.Score = models.NewNumber(0)
	if err := tx.Model(rec).Updates(map[string]interface{}{
		"status":        rec.Status,
		"unstake_block": gorm.Expr("NULL"),
		"score":         rec.Score.String(),
	}).Error; err != nil {
		return fmt.Errorf("revert unstake: update record error: %v", err)
	}
	return nil
}

// CalculateConsensusScoreByBlocks calculates score based on block duration: score = amt × durationBlocks
func (e *DBClient) CalculateConsensusScoreByBlocks(amt *models.Number, durationBlocks int64) *big.Int {
	if durationBlocks < 0 {
		durationBlocks = 0
	}
	if amt == nil || amt.Int().Sign() <= 0 || durationBlocks == 0 {
		return big.NewInt(0)
	}

	// Pure integer fixed-point: Score = amt * ln(1 + B), no floating point used
	const SCALE int64 = 1_000_000_000 // 1e9
	const Q uint = 32
	const LN2_SCALED int64 = 693147180 // ln(2) * SCALE

	n := new(big.Int).Add(big.NewInt(durationBlocks), big.NewInt(1))
	k := n.BitLen() - 1
	if k < 0 {
		k = 0
	}

	oneQ := new(big.Int).Lsh(big.NewInt(1), Q)
	aFixed := new(big.Int).Lsh(n, Q)
	if k > 0 {
		aFixed.Rsh(aFixed, uint(k))
	}
	if aFixed.Cmp(oneQ) < 0 {
		aFixed.Set(oneQ)
	}

	num := new(big.Int).Sub(aFixed, oneQ)
	den := new(big.Int).Add(aFixed, oneQ)
	if den.Sign() == 0 {
		return big.NewInt(0)
	}
	yFixed := new(big.Int).Lsh(num, Q)
	yFixed.Quo(yFixed, den)

	y2 := new(big.Int).Mul(yFixed, yFixed)
	y2.Rsh(y2, Q)
	y3 := new(big.Int).Mul(y2, yFixed)
	y3.Rsh(y3, Q)
	y5 := new(big.Int).Mul(y3, y2)
	y5.Rsh(y5, Q)

	term1 := new(big.Int).Set(yFixed)
	term3 := new(big.Int).Quo(y3, big.NewInt(3))
	term5 := new(big.Int).Quo(y5, big.NewInt(5))
	sumFixed := new(big.Int).Add(term1, term3)
	sumFixed.Add(sumFixed, term5)

	ln_a_scaled := new(big.Int).Mul(sumFixed, big.NewInt(2))
	ln_a_scaled.Mul(ln_a_scaled, big.NewInt(SCALE))
	ln_a_scaled.Quo(ln_a_scaled, oneQ)

	ln2k := big.NewInt(0).Mul(big.NewInt(int64(k)), big.NewInt(LN2_SCALED))
	ln_n_scaled := new(big.Int).Add(ln2k, ln_a_scaled)

	numScore := new(big.Int).Mul(amt.Int(), ln_n_scaled)
	numScore.Quo(numScore, big.NewInt(SCALE))

	return numScore
}

// CalculateConsensusScore calculates score based on start and end blocks
func (e *DBClient) CalculateConsensusScore(amt *models.Number, stakeBlock, endBlock int64) *big.Int {
	return e.CalculateConsensusScoreByBlocks(amt, endBlock-stakeBlock)
}

// 积分相关的复杂函数已废弃，改用“解押结算：score = amt × 持有区块数”的简单口径

// CalculateConsensusDecayedScore 计算解锁后的积分衰减值（按区块）
// 公式：Score(t) = Score0 × [β + (1 − β) × e^(−λ × t)]
// t 按区块计；λ、β 为参数（推荐 λ=0.1, β=0.2）
func (e *DBClient) CalculateConsensusDecayedScore(score0 *big.Int, tBlocks int64, lambda float64, beta float64) *big.Int {
	if score0 == nil || score0.Sign() <= 0 {
		return big.NewInt(0)
	}
	if tBlocks < 0 {
		tBlocks = 0
	}
	if beta < 0 {
		beta = 0
	}
	if beta > 1 {
		beta = 1
	}

	// 使用定点数计算衰减因子
	const SCALE int64 = 1_000_000_000 // 1e9
	decay := beta + (1.0-beta)*math.Exp(-lambda*float64(tBlocks))
	decayScaled := int64(decay * float64(SCALE))

	// 计算衰减后的积分：score0 * decay
	result := new(big.Int).Mul(score0, big.NewInt(decayScaled))
	result.Quo(result, big.NewInt(SCALE))

	return result
}

// CalculateConsensusDecayedScoreByBlocks 基于区块差计算衰减值（t 为区块数）
func (e *DBClient) CalculateConsensusDecayedScoreByBlocks(score0 *big.Int, unstakeBlock, currentBlock int64, blocksPerDay int64, lambda float64, beta float64) *big.Int {
	tBlocks := currentBlock - unstakeBlock
	if tBlocks < 0 {
		tBlocks = 0
	}
	return e.CalculateConsensusDecayedScore(score0, tBlocks, lambda, beta)
}

// GetConsensusRecordDecayedScore 基于单笔质押记录，计算当前衰减后的积分
// 仅对 status=closed 的记录生效；active 记录返回 0（尚未解锁）
func (e *DBClient) GetConsensusRecordDecayedScore(record *models.ConsensusStakeRecord, currentBlock int64, blocksPerDay int64, lambda float64, beta float64) *big.Int {
	if record == nil || record.Status != "closed" || record.UnstakeBlock == nil || record.Score == nil {
		return big.NewInt(0)
	}
	scoreInt := record.Score.Int()
	return e.CalculateConsensusDecayedScoreByBlocks(scoreInt, *record.UnstakeBlock, currentBlock, blocksPerDay, lambda, beta)
}
