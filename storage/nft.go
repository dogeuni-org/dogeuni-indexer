package storage

import (
	"dogeuni-indexer/models"
	"fmt"
	"gorm.io/gorm"
)

func (db *DBClient) NftDeploy(tx *gorm.DB, model *models.NftInfo) error {

	nftc := &models.NftCollect{
		Tick:          model.Tick,
		Total:         model.Total,
		Model:         model.Model,
		Prompt:        model.Prompt,
		Image:         model.Image,
		ImagePath:     model.ImagePath,
		HolderAddress: model.HolderAddress,
		DeployHash:    model.TxHash,
	}
	err := tx.Create(nftc).Error
	if err != nil {
		return fmt.Errorf("NftDeploy err: %s order_id: %s", err.Error(), model.OrderId)
	}

	return nil
}

func (db *DBClient) NftMint(tx *gorm.DB, model *models.NftInfo) error {
	//err := db.MintNft(tx, model.Tick, model.HolderAddress, model.Seed, model.Prompt, model.Image, model.ImagePath, model.TxHash, model.BlockNumber, false)
	//if err != nil {
	//	return fmt.Errorf("NftMint err: %s order_id: %s", err.Error(), model.OrderId)
	//}
	return nil
}

func (db *DBClient) NftTransfer(tx *gorm.DB, model *models.NftInfo) error {
	//err := db.TransferNft(tx, model.Tick, model.HolderAddress, model.ToAddress, model.TickId, model.BlockNumber, false)
	//if err != nil {
	//	return fmt.Errorf("NftTransfer err: %s order_id: %s", err.Error(), model.OrderId)
	//}
	//
	return nil
}
