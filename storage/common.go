package storage

import (
	"dogeuni-indexer/models"
	"errors"
	"fmt"
	"github.com/dogecoinw/go-dogecoin/log"
	"gorm.io/gorm"
	"math/big"
	"time"
)

func (db *DBClient) ScheduledTasks(height int64) error {

	s := time.Now()

	//if height < 5260645 {
	//	err = db.StakeUpdatePoolScheduled(tx, height)
	//	if err != nil {
	//		tx.Rollback()
	//		return err
	//	}
	//}

	tx := db.DB.Begin()
	err := db.BoxDeployScheduled(tx, height)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}

	log.Info("explorer", "StakeUpdatePool time", time.Now().Sub(s).String())
	return nil
}

func (db *DBClient) TransferDrc20(tx *gorm.DB, tick, from, to string, amt *big.Int, txHash string, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "Transfer", "start", "tick", tick, "from", from, "to", to, "amt", amt.String(), "fork", fork)

	if amt.Cmp(big.NewInt(0)) < 1 {
		return fmt.Errorf("transfer amt < 0")
	}

	if from == to {
		return fmt.Errorf("transfer from and to addresses are the same")
	}

	addFrom := &models.Drc20CollectAddress{}
	err := tx.Where("tick = ? and holder_address = ?", tick, from).First(addFrom).Error
	if err != nil {
		return fmt.Errorf("transfer err: %s tick: %s from : %s", err.Error(), tick, from)
	}

	if amt.Cmp(addFrom.AmtSum.Int()) > 0 {
		return fmt.Errorf("insufficient balance : %s tick: %s from : %s  transfer : %s", addFrom.AmtSum.String(), tick, from, amt.String())
	}

	addTo := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick, to).First(addTo).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("mint err: %s tick: %s to : %s", err.Error(), tick, to)
		}

		addTo.AmtSum = (*models.Number)(big.NewInt(0))
		addTo.Tick = tick
		addTo.HolderAddress = to
		err := tx.Create(addTo).Error
		if err != nil {
			return fmt.Errorf("mint err: %s tick: %s to : %s", err.Error(), tick, to)
		}
	}

	count1 := addFrom.AmtSum.Int()
	count2 := addTo.AmtSum.Int()

	sub := big.NewInt(0).Sub(count1, amt)
	add := big.NewInt(0).Add(count2, amt)

	err = tx.Model(addFrom).Where("tick = ? and holder_address = ?", tick, from).Update("amt_sum", sub.String()).Error
	if err != nil {
		return fmt.Errorf("transfer err: %s tick: %s from : %s", err.Error(), tick, from)
	}

	err = tx.Model(addTo).Where("tick = ? and holder_address = ?", tick, to).Update("amt_sum", add.String()).Error
	if err != nil {
		return fmt.Errorf("transfer err: %s tick: %s to : %s", err.Error(), tick, to)
	}

	if !fork {
		revert := &models.Drc20Revert{
			FromAddress: from,
			ToAddress:   to,
			Tick:        tick,
			Amt:         (*models.Number)(amt),
			TxHash:      txHash,
			BlockNumber: height,
		}
		err = tx.Create(revert).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBClient) MintDrc20(tx *gorm.DB, tick, holderAddress string, amt *big.Int, txHash string, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "Mint", "start", "tick", tick, "holderAddress", holderAddress, "amt", amt.String())

	drc20c := &models.Drc20Collect{}
	err := tx.Where("tick = ?", tick).First(drc20c).Error
	if err != nil {
		return fmt.Errorf("Mint FindDrc20InfoByTick err: %s tick: %s", err.Error(), tick)
	}

	drc20ca := &models.Drc20CollectAddress{}

	err = tx.Where("tick = ? and holder_address = ?", tick, holderAddress).First(drc20ca).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("mint FindDrc20AddressInfoByTick err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
		}

		drc20ca.AmtSum = (*models.Number)(big.NewInt(0))
		drc20ca.Tick = tick
		drc20ca.HolderAddress = holderAddress
		err := tx.Create(drc20ca).Error
		if err != nil {
			return fmt.Errorf("mint CreateAddressBalanceMint err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
		}
	}

	count := drc20c.AmtSum.Int()
	count1 := drc20ca.AmtSum.Int()

	sum := big.NewInt(0).Add(count, amt)
	sum1 := big.NewInt(0).Add(count1, amt)

	trans := drc20c.Transactions + 1
	if fork {
		trans = drc20c.Transactions - 1
		if trans < 0 {
			trans = 0
		}
	}

	err = tx.Model(drc20c).Where("tick = ?", tick).Updates(map[string]interface{}{"amt_sum": sum.String(), "transactions": trans}).Error
	if err != nil {
		return fmt.Errorf("mint UpdateDrc20InfoMint err: %s tick: %s", err.Error(), tick)
	}

	err = tx.Model(drc20ca).Where("tick = ? and holder_address = ?", tick, holderAddress).Update("amt_sum", sum1.String()).Error
	if err != nil {
		return fmt.Errorf("mint UpdateAddressBalanceMint err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
	}

	if !fork {
		revert := &models.Drc20Revert{
			ToAddress:   holderAddress,
			Tick:        tick,
			Amt:         (*models.Number)(amt),
			TxHash:      txHash,
			BlockNumber: height,
		}
		err = tx.Create(revert).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBClient) BurnDrc20(tx *gorm.DB, tick, holderAddress string, amt *big.Int, txHash string, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "Burn", "start", "tick", tick, "holderAddress", holderAddress, "amt", amt.String())

	drc20c := &models.Drc20Collect{}
	err := tx.Where("tick = ?", tick).First(drc20c).Error
	if err != nil {
		return fmt.Errorf("burn err: %s tick: %s", err.Error(), tick)
	}

	drc20ca := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick, holderAddress).First(drc20ca).Error
	if err != nil {
		return fmt.Errorf("burn err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
	}

	count := drc20c.AmtSum.Int()
	count1 := drc20ca.AmtSum.Int()

	if count.Cmp(amt) == -1 {
		return fmt.Errorf("burn count < amount tick: %s count: %s amount: %s", tick, count.String(), amt.String())
	}

	if count1.Cmp(amt) == -1 {
		return fmt.Errorf("burn count1 < amount tick: %s count1: %s amount: %s", tick, count1.String(), amt.String())
	}

	sum := big.NewInt(0).Sub(count, amt)
	sum1 := big.NewInt(0).Sub(count1, amt)

	trans := drc20c.Transactions + 1
	if fork {
		trans = drc20c.Transactions - 1
		if trans < 0 {
			trans = 0
		}
	}

	err = tx.Model(drc20c).Where("tick = ?", tick).Updates(map[string]interface{}{"amt_sum": sum.String(), "transactions": trans}).Error
	if err != nil {
		return fmt.Errorf("mint UpdateDrc20InfoMint err: %s tick: %s", err.Error(), tick)
	}

	err = tx.Model(drc20ca).Where("tick = ? and holder_address = ?", tick, holderAddress).Update("amt_sum", sum1.String()).Error
	if err != nil {
		return fmt.Errorf("mint UpdateAddressBalanceMint err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
	}

	if !fork {
		revert := &models.Drc20Revert{
			FromAddress: holderAddress,
			Tick:        tick,
			Amt:         (*models.Number)(amt),
			TxHash:      txHash,
			BlockNumber: height,
		}
		err = tx.Create(revert).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBClient) TransferFile(tx *gorm.DB, from, to string, fileId string, txHash string, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "TransferFile", "start", "from", from, "to", to, "fileId", fileId, "fork", fork)

	err := tx.Model(&models.FileCollectAddress{}).Where("file_id = ? AND holder_address = ?", fileId, from).Update("holder_address", to).Error
	if err != nil {
		return err
	}

	if !fork {
		revert := &models.FileRevert{
			FromAddress: from,
			ToAddress:   to,
			FileId:      fileId,
			TxHash:      txHash,
			BlockNumber: height,
		}
		err = tx.Create(revert).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBClient) StakeStakeV1(tx *gorm.DB, tick, holderAddress string, amt *big.Int, txHash string, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "stake", "start", "tick", tick, "holderAddress", holderAddress, "amt", amt.String())

	stakec := &models.StakeCollect{}

	err := tx.Where("tick = ?", tick).First(stakec).Error
	if err != nil {
		return fmt.Errorf("StakeStake FindStakeCollectByTick err: %s tick: %s", err.Error(), tick)
	}

	amt0 := big.NewInt(0).Add(stakec.Amt.Int(), amt)
	err = tx.Model(stakec).Where("tick = ?", tick).Update("amt", amt0.String()).Error
	if err != nil {
		return fmt.Errorf("StakeStake UpdateStakeCollect err: %s tick: %s", err.Error(), tick)
	}

	stakeca := &models.StakeCollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick, holderAddress).First(stakeca).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			stakeca.Amt = (*models.Number)(amt)
			stakeca.Tick = tick
			stakeca.HolderAddress = holderAddress
			stakeca.Reward = (*models.Number)(big.NewInt(0))
			err := tx.Create(stakeca).Error
			if err != nil {
				return fmt.Errorf("StakeStake CreateStakeCollectAddress err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
			}
		} else {
			return fmt.Errorf("StakeStake FindStakeCollectAddress err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
		}
	} else {
		amt1 := big.NewInt(0).Add(stakeca.Amt.Int(), amt)
		err = tx.Model(stakeca).Where("tick = ? and holder_address = ?", tick, holderAddress).Update("amt", amt1.String()).Error
		if err != nil {
			return fmt.Errorf("StakeStake UpdateStakeCollectAddress err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
		}
	}

	if !fork {
		sr := &models.StakeRevert{
			Tick:        tick,
			ToAddress:   holderAddress,
			Amt:         (*models.Number)(amt),
			TxHash:      txHash,
			BlockNumber: height,
		}

		err = tx.Create(sr).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBClient) StakeUnStakeV1(tx *gorm.DB, tick, holderAddress string, amt *big.Int, txHash string, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "unstake", "start", "tick", tick, "holderAddress", holderAddress, "amt", amt.String())

	stakec := &models.StakeCollect{}
	err := tx.Where("tick = ?", tick).First(stakec).Error
	if err != nil {
		return fmt.Errorf("StakeStake FindStakeCollectByTick err: %s tick: %s", err.Error(), tick)
	}

	amt0 := big.NewInt(0).Sub(stakec.Amt.Int(), amt)
	if amt0.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("StakeStake amt0 < 0 tick: %s", tick)
	}

	err = tx.Model(stakec).Where("tick = ?", tick).Update("amt", amt0.String()).Error
	if err != nil {
		return fmt.Errorf("StakeStake UpdateStakeCollect err: %s tick: %s", err.Error(), tick)
	}

	stakeca := &models.StakeCollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick, holderAddress).First(stakeca).Error
	if err != nil {
		return fmt.Errorf("StakeStake FindStakeCollectAddress err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
	}

	amt1 := big.NewInt(0).Sub(stakeca.Amt.Int(), amt)
	if amt1.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("StakeStake amt1 < 0 err: %s tick: %s", err, tick)
	}

	err = tx.Model(stakeca).Where("tick = ? and holder_address = ?", tick, holderAddress).Update("amt", amt1.String()).Error
	if err != nil {
		return fmt.Errorf("StakeStake UpdateStakeCollectAddress err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
	}

	if !fork {
		sr := &models.StakeRevert{
			Tick:        tick,
			FromAddress: holderAddress,
			Amt:         (*models.Number)(amt),
			TxHash:      txHash,
			BlockNumber: height,
		}
		err = tx.Create(sr).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBClient) StakeRewardV1(tx *gorm.DB, tick, holderAddress string, txHash string, height int64) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	stakeca := &models.StakeCollectAddress{}
	err := tx.Where("tick = ? and holder_address = ?", tick, holderAddress).First(stakeca).Error
	if err != nil {
		return fmt.Errorf("StakeStake FindStakeCollectAddress err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
	}

	err = tx.Model(stakeca).Where("tick = ? and holder_address = ?", tick, holderAddress).Update("received_reward", stakeca.Reward).Error
	if err != nil {
		return fmt.Errorf("StakeStake UpdateStakeCollectAddress err: %s tick: %s from : %s", err.Error(), tick, holderAddress)
	}

	reward := big.NewInt(0).Sub(stakeca.Reward.Int(), stakeca.ReceivedReward.Int())

	rsr := &models.StakeRewardRevert{
		Tick:        tick,
		ToAddress:   holderAddress,
		Amt:         (*models.Number)(reward),
		TxHash:      txHash,
		BlockNumber: height,
	}

	err = tx.Create(rsr).Error
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) StakeGetRewardV1(tx *gorm.DB, holderAddress, tick string) ([]*models.HolderReward, error) {

	poolResults := make([]*models.Drc20CollectAddress, 0)
	err := tx.Where("holder_address = ? and amt_sum != '0'", stakePoolAddress).Find(&poolResults).Error
	if err != nil {
		return nil, err
	}

	stakeAddressCollect := &models.StakeCollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick, holderAddress).First(stakeAddressCollect).Error
	if err != nil {
		return nil, err
	}

	rewards := make([]*models.HolderReward, 0)
	reward := big.NewInt(0).Sub(stakeAddressCollect.Reward.Int(), stakeAddressCollect.ReceivedReward.Int())

	if tick == "UNIX-SWAP-WDOGE(WRAPPED-DOGE)" {
		rewards = append(rewards, &models.HolderReward{
			Tick:   "WDOGE(WRAPPED-DOGE)",
			Reward: reward,
		})
		return rewards, nil
	}

	unixPool := &models.Drc20CollectAddress{}
	err = tx.Where("tick = 'UNIX' and holder_address = ? and amt_sum != '0'", stakePoolAddress).Find(&unixPool).Error
	if err != nil {
		return nil, err
	}

	for _, ar := range poolResults {
		if ar.Tick == "UNIX" || ar.Tick == "WDOGE(WRAPPED-DOGE)" {
			continue
		}

		amt := big.NewInt(0).Div(big.NewInt(0).Mul(ar.AmtSum.Int(), reward), unixPool.AmtSum.Int())
		receivedAmt := big.NewInt(0).Div(big.NewInt(0).Mul(ar.AmtSum.Int(), stakeAddressCollect.ReceivedReward.Int()), unixPool.AmtSum.Int())
		TotalAmt := big.NewInt(0).Div(big.NewInt(0).Mul(ar.AmtSum.Int(), stakeAddressCollect.Reward.Int()), unixPool.AmtSum.Int())
		TotalAmt = big.NewInt(0).Add(TotalAmt, amt)

		rewards = append(rewards, &models.HolderReward{
			Tick:            ar.Tick,
			Reward:          amt,
			ReceivedReward:  receivedAmt,
			TotalRewardPool: TotalAmt,
		})
	}

	rewards = append(rewards, &models.HolderReward{
		Tick:            "UNIX",
		Reward:          reward,
		ReceivedReward:  stakeAddressCollect.ReceivedReward.Int(),
		TotalRewardPool: big.NewInt(0).Add(stakeAddressCollect.Reward.Int(), reward),
	})

	return rewards, nil
}

func (db *DBClient) BoxDeployScheduled(tx *gorm.DB, height int64) error {

	bcs := make([]*models.BoxCollect, 0)
	err := tx.Where("liqblock = ? and is_del = 0", height).Find(&bcs).Error
	if err != nil {
		return err
	}

	for _, bc := range bcs {
		if bc.LiqAmtFinish.Int().Cmp(big.NewInt(0)) > 0 {
			err = db.BoxFinish(tx, bc, height)
			if err != nil {
				return err
			}
		} else {
			err = db.BoxRefund(tx, bc, height)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (db *DBClient) StakeGetRewardV2(stakeId, holderAddress string, height int64) (*models.Number, error) {

	stakea := &models.StakeV2CollectAddress{}
	err := db.DB.Where("stake_id = ? AND holder_address = ?", stakeId, holderAddress).First(stakea).Error
	if err != nil {
		return nil, err
	}

	stake := &models.StakeV2Collect{}
	err = db.DB.Where("stake_id = ?", stakeId).First(stake).Error
	if err != nil {
		return nil, err
	}

	if stake.TotalStaked.Int().Cmp(big.NewInt(0)) <= 0 {
		return models.NewNumber(0), nil
	}

	blocksPassed := height - stake.LastRewardBlock
	reward := big.NewInt(0).Mul(big.NewInt(blocksPassed), stake.EachReward.Int())
	accRewardPerShare := big.NewInt(0).Div(big.NewInt(0).Mul(reward, big.NewInt(1e8)), stake.TotalStaked.Int())
	stake.AccRewardPerShare = (*models.Number)(big.NewInt(0).Add(stake.AccRewardPerShare.Int(), accRewardPerShare))

	pending := big.NewInt(0).Div(big.NewInt(0).Mul(stakea.Amt.Int(), stake.AccRewardPerShare.Int()), big.NewInt(1e8))
	pending = big.NewInt(0).Sub(pending, stakea.RewardDebt.Int())
	if big.NewInt(0).Cmp(pending) < 0 {
		pending = big.NewInt(0).Add(pending, stakea.PendingReward.Int())
	}

	return (*models.Number)(pending), nil
}

func (db *DBClient) TransferMeme20(tx *gorm.DB, tickId, from, to string, amt *big.Int, txHash string, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "Transfer", "start", "tickId", tickId, "from", from, "to", to, "amt", amt.String(), "fork", fork)

	if amt.Cmp(big.NewInt(0)) < 1 {
		return fmt.Errorf("transfer amt < 0")
	}

	if from == to {
		return fmt.Errorf("transfer from and to addresses are the same")
	}

	addFrom := &models.Meme20CollectAddress{}
	err := tx.Where("tick_id = ? and holder_address = ?", tickId, from).First(addFrom).Error
	if err != nil {
		return fmt.Errorf("transfer err: %s tickId: %s from: %s", err.Error(), tickId, from)
	}

	if amt.Cmp(addFrom.Amt.Int()) > 0 {
		return fmt.Errorf("insufficient balance: %s tickId: %s from: %s transfer: %s", addFrom.Amt.String(), tickId, from, amt.String())
	}

	addTo := &models.Meme20CollectAddress{}
	err = tx.Where("tick_id = ? and holder_address = ?", tickId, to).First(addTo).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("transfer err: %s tickId: %s to : %s", err.Error(), tickId, to)
		}

		addTo.Amt = (*models.Number)(big.NewInt(0))
		addTo.TickId = tickId
		addTo.HolderAddress = to
		err := tx.Create(addTo).Error
		if err != nil {
			return fmt.Errorf("mint err: %s tickId: %s to : %s", err.Error(), tickId, to)
		}
	}

	count1 := addFrom.Amt.Int()
	count2 := addTo.Amt.Int()

	sub := big.NewInt(0).Sub(count1, amt)
	add := big.NewInt(0).Add(count2, amt)

	err = tx.Model(addFrom).Where("tick_id = ? and holder_address = ?", tickId, from).Update("amt", sub.String()).Error
	if err != nil {
		return fmt.Errorf("transfer err: %s tick: %s from : %s", err.Error(), tickId, from)
	}

	err = tx.Model(addTo).Where("tick_id = ? and holder_address = ?", tickId, to).Update("amt", add.String()).Error
	if err != nil {
		return fmt.Errorf("transfer err: %s tick: %s to : %s", err.Error(), tickId, to)
	}

	mc := &models.Meme20Collect{}
	err = tx.Where("tick_id = ?", tickId).First(mc).Error
	if err != nil {
		return fmt.Errorf("transfer err: %s tick: %s", err.Error(), tickId)
	}

	trans := mc.Transactions + 1
	if fork {
		trans = mc.Transactions - 1
		if trans < 0 {
			trans = 0
		}
	}

	err = tx.Model(mc).Where("tick_id = ?", tickId).Update("transactions", trans).Error
	if err != nil {
		return fmt.Errorf("transfer err: %s tick: %s", err.Error(), tickId)
	}

	if !fork {
		revert := &models.Meme20Revert{
			FromAddress: from,
			ToAddress:   to,
			TickId:      tickId,
			Amt:         (*models.Number)(amt),
			TxHash:      txHash,
			BlockNumber: height,
		}
		err = tx.Create(revert).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBClient) MintMeme20(tx *gorm.DB, tickId, holderAddress string, amt *big.Int, txHash string, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "Mint meme", "start", "tickId", tickId, "holderAddress", holderAddress, "amt", amt.String())

	meme20c := &models.Meme20Collect{}
	err := tx.Where("tick_id = ?", tickId).First(meme20c).Error
	if err != nil {
		return fmt.Errorf("mint meme err: %s tickId: %s", err.Error(), tickId)
	}

	meme20ca := &models.Meme20CollectAddress{}

	err = tx.Where("tick_id = ? and holder_address = ?", tickId, holderAddress).First(meme20ca).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("mint meme err: %s tickId: %s from : %s", err.Error(), tickId, holderAddress)
		}

		meme20ca.Amt = (*models.Number)(big.NewInt(0))
		meme20ca.TickId = tickId
		meme20ca.HolderAddress = holderAddress
		err := tx.Create(meme20ca).Error
		if err != nil {
			return fmt.Errorf("mint meme err: %s tickId: %s from : %s", err.Error(), tickId, holderAddress)
		}
	}

	count1 := meme20ca.Amt.Int()

	sum1 := big.NewInt(0).Add(count1, amt)

	trans := meme20ca.Transactions + 1
	if fork {
		trans = meme20ca.Transactions - 1
		if trans < 0 {
			trans = 0
		}
	}

	err = tx.Model(meme20ca).Where("tick_id = ? and holder_address = ?", tickId, holderAddress).Updates(
		map[string]interface{}{
			"amt":          sum1.String(),
			"transactions": trans,
		}).Error
	if err != nil {
		return fmt.Errorf("mint meme err: %s tickId: %s from : %s", err.Error(), tickId, holderAddress)
	}

	count2 := meme20c.Max.Int()
	max0 := big.NewInt(0).Add(count2, amt)

	transc := meme20c.Transactions + 1
	if fork {
		transc = meme20c.Transactions - 1
		if transc < 0 {
			transc = 0
		}
	}

	err = tx.Model(meme20c).Where("tick_id = ?", tickId).Updates(
		map[string]interface{}{
			"max_":         max0.String(),
			"transactions": transc,
		}).Error

	if !fork {
		revert := &models.Meme20Revert{
			ToAddress:   holderAddress,
			TickId:      tickId,
			Amt:         (*models.Number)(amt),
			TxHash:      txHash,
			BlockNumber: height,
		}
		err = tx.Create(revert).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBClient) BurnMeme20(tx *gorm.DB, tickId, holderAddress string, amt *big.Int, txHash string, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "Burn meme", "start", "tickId", tickId, "holderAddress", holderAddress, "amt", amt.String())

	meme20ca := &models.Meme20CollectAddress{}
	err := tx.Where("tick_id = ? and holder_address = ?", tickId, holderAddress).First(meme20ca).Error
	if err != nil {
		return fmt.Errorf("burn meme err: %s tickId: %s from : %s", err.Error(), tickId, holderAddress)

	}

	count1 := meme20ca.Amt.Int()

	if count1.Cmp(amt) == -1 {
		return fmt.Errorf("burn count1 < amount tickId: %s count1: %s amount: %s", tickId, count1.String(), amt.String())
	}

	sum1 := big.NewInt(0).Sub(count1, amt)

	trans := meme20ca.Transactions - 1
	if fork {
		trans = meme20ca.Transactions + 1
		if trans < 0 {
			trans = 0
		}
	}

	err = tx.Model(meme20ca).Where("tick_id = ? and holder_address = ?", tickId, holderAddress).Updates(
		map[string]interface{}{
			"amt":          sum1.String(),
			"transactions": trans,
		}).Error

	if err != nil {
		return fmt.Errorf("burn meme err: %s tick: %s from : %s", err.Error(), tickId, holderAddress)
	}

	meme20c := &models.Meme20Collect{}
	err = tx.Where("tick_id = ?", tickId).First(meme20c).Error
	if err != nil {
		return fmt.Errorf("burn meme err: %s tickId: %s", err.Error(), tickId)
	}

	count2 := meme20c.Max.Int()
	if count2.Cmp(amt) == -1 {
		return fmt.Errorf("burn count2 < amount tickId: %s count2: %s amount: %s", tickId, count2.String(), amt.String())
	}

	max0 := big.NewInt(0).Sub(count2, amt)

	transc := meme20c.Transactions + 1
	if fork {
		transc = meme20c.Transactions - 1
		if transc < 0 {
			transc = 0
		}
	}

	err = tx.Model(meme20c).Where("tick_id = ?", tickId).Updates(
		map[string]interface{}{
			"max_":         max0.String(),
			"transactions": transc,
		}).Error

	if err != nil {
		return fmt.Errorf("burn meme err: %s tickId: %s", err.Error(), tickId)
	}

	if !fork {
		revert := &models.Meme20Revert{
			FromAddress: holderAddress,
			TickId:      tickId,
			Amt:         (*models.Number)(amt),
			TxHash:      txHash,
			BlockNumber: height,
		}
		err = tx.Create(revert).Error
		if err != nil {
			return err
		}
	}

	return nil
}
