package storage

import (
	"dogeuni-indexer/models"
	"fmt"
	"github.com/dogecoinw/go-dogecoin/log"
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

	revert := &models.NftRevert{
		Op:          "deploy",
		Tick:        model.Tick,
		ToAddress:   model.HolderAddress,
		BlockNumber: model.BlockNumber,
	}
	err = tx.Create(revert).Error
	if err != nil {
		return fmt.Errorf("NftDeploy revert err: %s order_id: %s", err.Error(), model.OrderId)
	}

	return nil
}

func (db *DBClient) NftMint(tx *gorm.DB, model *models.NftInfo) error {
	tickId, err := db.MintNft(tx, model.Tick, model.HolderAddress, model.Prompt, model.ImagePath, model.TxHash, model.BlockNumber, false)
	if err != nil {
		return fmt.Errorf("NftMint err: %s order_id: %s", err.Error(), model.OrderId)
	}

	// the token id is assigned sequentially at mint time; record it on the inscription row
	model.TickId = tickId
	err = tx.Model(&models.NftInfo{}).Where("tx_hash = ?", model.TxHash).Update("tick_id", tickId).Error
	if err != nil {
		return fmt.Errorf("NftMint update tick_id err: %s order_id: %s", err.Error(), model.OrderId)
	}
	return nil
}

func (db *DBClient) NftTransfer(tx *gorm.DB, model *models.NftInfo) error {
	err := db.TransferNft(tx, model.Tick, model.HolderAddress, model.ToAddress, model.TickId, model.BlockNumber, false)
	if err != nil {
		return fmt.Errorf("NftTransfer err: %s order_id: %s", err.Error(), model.OrderId)
	}
	return nil
}

// MintNft assigns the next token id of tick to holder and returns it
func (db *DBClient) MintNft(tx *gorm.DB, tick, holder, prompt, imagePath, txHash string, height int64, fork bool) (int64, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "MintNft", "start", "tick", tick, "holder", holder)

	err := tx.Model(&models.NftCollect{}).Where("tick = ?", tick).Updates(map[string]interface{}{
		"transactions": gorm.Expr("transactions + 1"),
		"tick_sum":     gorm.Expr("tick_sum + 1"),
	}).Error
	if err != nil {
		return 0, err
	}

	nftc := &models.NftCollect{}
	err = tx.Where("tick = ?", tick).First(nftc).Error
	if err != nil {
		return 0, err
	}
	tickId := nftc.TickSum

	nfca := &models.NftCollectAddress{
		Tick:          tick,
		TickId:        tickId,
		Prompt:        prompt,
		ImagePath:     imagePath,
		DeployHash:    txHash,
		HolderAddress: holder,
	}
	err = tx.Create(nfca).Error
	if err != nil {
		return 0, err
	}

	if !fork {
		revert := &models.NftRevert{
			Op:          "mint",
			Tick:        tick,
			TickId:      tickId,
			ToAddress:   holder,
			BlockNumber: height,
		}
		err = tx.Create(revert).Error
		if err != nil {
			return 0, err
		}
	}

	return tickId, nil
}

func (db *DBClient) TransferNft(tx *gorm.DB, tick, from, to string, tickId int64, height int64, fork bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	log.Info("explorer", "TransferNft", "start", "tick", tick, "from", from, "to", to, "tickId", tickId, "fork", fork)

	delta := "transactions + 1"
	if fork {
		delta = "transactions - 1"
	}
	err := tx.Model(&models.NftCollect{}).Where("tick = ?", tick).Update("transactions", gorm.Expr(delta)).Error
	if err != nil {
		return err
	}

	res := tx.Model(&models.NftCollectAddress{}).Where("tick = ? AND tick_id = ? AND holder_address = ?", tick, tickId, from).Update("holder_address", to)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return fmt.Errorf("TransferNft %s #%d not held by %s", tick, tickId, from)
	}

	if !fork {
		revert := &models.NftRevert{
			Op:          "transfer",
			Tick:        tick,
			TickId:      tickId,
			FromAddress: from,
			ToAddress:   to,
			BlockNumber: height,
		}
		err = tx.Create(revert).Error
		if err != nil {
			return err
		}
	}

	return nil
}
