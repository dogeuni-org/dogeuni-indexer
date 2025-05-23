package storage

import (
	"dogeuni-indexer/models"
	"fmt"
	"gorm.io/gorm"
)

func (db *DBClient) FileDeploy(tx *gorm.DB, file *models.FileInfo) error {

	fileCollectAddress := &models.FileCollectAddress{
		FileId:        file.FileId,
		FilePath:      file.FilePath,
		HolderAddress: file.HolderAddress,
	}

	err := tx.Create(fileCollectAddress).Error
	if err != nil {
		return fmt.Errorf("deploy InstallNftCollect err: %s order_id: %s", err.Error(), file.OrderId)
	}

	revert := &models.FileRevert{
		FromAddress: "",
		ToAddress:   file.ToAddress,
		FileId:      file.FileId,
		BlockNumber: file.BlockNumber,
	}

	err = tx.Create(revert).Error
	if err != nil {
		return fmt.Errorf("deploy InstallNftRevert err: %s order_id: %s", err.Error(), file.OrderId)
	}

	return nil
}

func (db *DBClient) FileTransfer(tx *gorm.DB, file *models.FileInfo) error {
	err := db.TransferFile(tx, file.HolderAddress, file.ToAddress, file.FileId, file.TxHash, file.BlockNumber, false)
	if err != nil {
		return fmt.Errorf("transfer err: %s order_id: %s", err, file.OrderId)
	}

	return nil
}
