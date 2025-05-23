package storage

import (
	"dogeuni-indexer/models"
	"gorm.io/gorm"
	"math/big"
)

func (db *DBClient) ExchangeCreate(tx *gorm.DB, ex *models.ExchangeInfo, reservesAddress string) error {

	ec := &models.ExchangeCollect{
		ExId:            ex.ExId,
		Tick0:           ex.Tick0,
		Tick1:           ex.Tick1,
		Amt0:            ex.Amt0,
		Amt1:            ex.Amt1,
		HolderAddress:   ex.HolderAddress,
		ReservesAddress: reservesAddress,
	}

	err := tx.Save(ec).Error
	if err != nil {
		return err
	}

	err = db.TransferDrc20(tx, ex.Tick0, ex.HolderAddress, reservesAddress, ex.Amt0.Int(), ex.TxHash, ex.BlockNumber, false)
	if err != nil {
		return err
	}

	exr := &models.ExchangeRevert{
		Op:     "create",
		ExId:   ex.ExId,
		TxHash: ex.TxHash,
	}

	err = tx.Save(exr).Error
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) ExchangeTrade(tx *gorm.DB, ex *models.ExchangeInfo) error {

	exc := &models.ExchangeCollect{}
	err := tx.Where("ex_id = ?", ex.ExId).First(exc).Error
	if err != nil {
		return err
	}

	amt0 := new(big.Int).Sub(exc.Amt0.Int(), exc.Amt0Finish.Int())

	amt0Out := new(big.Int).Mul(ex.Amt1.Int(), exc.Amt0.Int())
	amt0Out = new(big.Int).Div(amt0Out, exc.Amt1.Int())

	if amt0.Cmp(amt0Out) < 0 {
		amt0Out = amt0
	}

	amt0Finish := new(big.Int).Add(exc.Amt0Finish.Int(), amt0Out)
	amt1Finish := new(big.Int).Add(exc.Amt1Finish.Int(), ex.Amt1.Int())

	tx.Model(&models.ExchangeCollect{}).Where("ex_id = ?", ex.ExId).Updates(map[string]interface{}{"amt0_finish": amt0Finish.String(), "amt1_finish": amt1Finish.String()})

	exr := &models.ExchangeRevert{
		Op:     "trade",
		ExId:   ex.ExId,
		Amt0:   (*models.Number)(amt0Out),
		Amt1:   ex.Amt1,
		TxHash: ex.TxHash,
	}

	err = tx.Save(exr).Error
	if err != nil {
		return err
	}

	ex.Tick0 = exc.Tick0
	ex.Tick1 = exc.Tick1
	ex.Amt0 = (*models.Number)(amt0Out)

	err = db.TransferDrc20(tx, exc.Tick1, ex.HolderAddress, exc.HolderAddress, ex.Amt1.Int(), ex.TxHash, ex.BlockNumber, false)
	if err != nil {
		return err
	}

	err = db.TransferDrc20(tx, exc.Tick0, exc.ReservesAddress, ex.HolderAddress, amt0Out, ex.TxHash, ex.BlockNumber, false)
	if err != nil {
		return err
	}

	return nil

}

func (db *DBClient) ExchangeCancel(tx *gorm.DB, ex *models.ExchangeInfo) error {

	exc := &models.ExchangeCollect{}
	err := tx.Where("ex_id = ?", ex.ExId).First(exc).Error
	if err != nil {
		return err
	}

	err = db.TransferDrc20(tx, exc.Tick0, exc.ReservesAddress, ex.HolderAddress, ex.Amt0.Int(), ex.TxHash, ex.BlockNumber, false)
	if err != nil {
		return err
	}

	amt0Finish := new(big.Int).Add(exc.Amt0Finish.Int(), ex.Amt0.Int())

	err = tx.Model(&models.ExchangeCollect{}).Where("ex_id = ?", ex.ExId).Update("amt0_finish", amt0Finish.String()).Error
	if err != nil {
		return err
	}

	ex.Tick0 = exc.Tick0
	ex.Tick1 = exc.Tick1

	exr := &models.ExchangeRevert{
		Op:     "cancel",
		ExId:   ex.ExId,
		Amt0:   ex.Amt0,
		TxHash: ex.TxHash,
	}

	err = tx.Save(exr).Error
	if err != nil {
		return err
	}

	return nil
}
