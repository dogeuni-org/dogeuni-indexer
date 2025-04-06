package storage

import (
	"dogeuni-indexer/models"
	"gorm.io/gorm"
)

func (db *DBClient) DogeDeposit(tx *gorm.DB, wdoge *models.WDogeInfo) error {

	err := db.MintDrc20(tx, wdoge.Tick, wdoge.HolderAddress, wdoge.Amt.Int(), wdoge.TxHash, wdoge.BlockNumber, false)
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) DogeWithdraw(tx *gorm.DB, wdoge *models.WDogeInfo) error {

	err := db.BurnDrc20(tx, wdoge.Tick, wdoge.HolderAddress, wdoge.Amt.Int(), wdoge.TxHash, wdoge.BlockNumber, false)
	if err != nil {
		return err
	}

	return nil
}
