package storage

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/utils"
	"gorm.io/gorm"
)

func (db *DBClient) CrossDeploy(tx *gorm.DB, cross *models.CrossInfo) error {

	//WDOGE(WRAPPED-DOGE)
	tick := "W" + cross.Tick + "(WRAPPED-" + cross.Tick + ")"
	drc20c := &models.Drc20Collect{
		Tick:          tick,
		Max:           (*models.Number)(utils.MAX_NUMBER),
		Dec:           8,
		HolderAddress: cross.HolderAddress,
		TxHash:        cross.TxHash,
	}

	err := tx.Create(drc20c).Error
	if err != nil {
		return err
	}

	cc := &models.CrossCollect{
		Tick:          tick,
		AdminAddress:  cross.AdminAddress,
		HolderAddress: cross.HolderAddress,
	}

	err = tx.Create(cc).Error
	if err != nil {
		return err
	}

	revert := &models.CrossRevert{
		Op:          "deploy",
		Tick:        tick,
		BlockNumber: cross.BlockNumber,
	}

	err = tx.Create(revert).Error
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) CrossMint(tx *gorm.DB, cross *models.CrossInfo) error {

	err := db.MintDrc20(tx, cross.Tick, cross.ToAddress, cross.Amt.Int(), cross.TxHash, cross.BlockNumber, false)
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) CrossBurn(tx *gorm.DB, cross *models.CrossInfo) error {
	err := db.BurnDrc20(tx, cross.Tick, cross.HolderAddress, cross.Amt.Int(), cross.TxHash, cross.BlockNumber, false)
	if err != nil {
		return err
	}
	return nil
}
