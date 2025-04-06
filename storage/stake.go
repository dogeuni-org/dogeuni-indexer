package storage

import (
	"dogeuni-indexer/models"
	"gorm.io/gorm"
)

func (db *DBClient) StakeStake(tx *gorm.DB, stake *models.StakeInfo, reservesAddress string) error {

	err := db.TransferDrc20(tx, stake.Tick, stake.HolderAddress, reservesAddress, stake.Amt.Int(), stake.TxHash, stake.BlockNumber, false)
	if err != nil {
		return err
	}

	err = db.StakeStakeV1(tx, stake.Tick, stake.HolderAddress, stake.Amt.Int(), stake.TxHash, stake.BlockNumber, false)
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) StakeUnStake(tx *gorm.DB, stake *models.StakeInfo, reservesAddress string) error {

	err := db.TransferDrc20(tx, stake.Tick, reservesAddress, stake.HolderAddress, stake.Amt.Int(), stake.TxHash, stake.BlockNumber, false)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = db.StakeUnStakeV1(tx, stake.Tick, stake.HolderAddress, stake.Amt.Int(), stake.TxHash, stake.BlockNumber, false)
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (db *DBClient) StakeGetReward(tx *gorm.DB, stake *models.StakeInfo) error {

	rewards, err := db.StakeGetRewardV1(tx, stake.HolderAddress, stake.Tick)
	if err != nil {
		return err
	}

	for _, reward := range rewards {
		err = db.TransferDrc20(tx, reward.Tick, stakePoolAddress, stake.HolderAddress, reward.Reward, stake.TxHash, stake.BlockNumber, false)
		if err != nil {
			return err
		}

		sri := &models.StakeRewardInfo{
			OrderId:     stake.OrderId,
			Tick:        reward.Tick,
			Amt:         (*models.Number)(reward.Reward),
			FromAddress: stakePoolAddress,
			ToAddress:   stake.HolderAddress,
			BlockNumber: stake.BlockNumber,
		}

		err = tx.Create(sri).Error
		if err != nil {
			return err
		}
	}

	err = db.StakeRewardV1(tx, stake.Tick, stake.HolderAddress, stake.TxHash, stake.BlockNumber)
	if err != nil {
		return err
	}

	return nil

}
