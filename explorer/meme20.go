package explorer

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/chaincfg/chainhash"
	"github.com/dogecoinw/go-dogecoin/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (e *Explorer) meme20Decode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.Meme20Info, error) {

	err := e.dbc.DB.Where("tx_hash = ?", tx.Hash).First(&models.Meme20Info{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("meme20 already exist or err %s", tx.Hash)
	}

	param := &models.Meme20Inscription{}
	err = json.Unmarshal(pushedData, param)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	meme, err := utils.ConvertMeme(param)
	if err != nil {
		return nil, fmt.Errorf("ConvetMeme err: %s", err.Error())
	}

	meme.OrderId = uuid.New().String()
	meme.FeeTxHash = tx.Vin[0].Txid

	meme.TxHash = tx.Hash
	meme.BlockHash = tx.BlockHash
	meme.BlockNumber = number
	meme.OrderStatus = 1

	txhash0, _ := chainhash.NewHashFromStr(tx.Vin[0].Txid)
	txRawResult0, err := e.node.GetRawTransactionVerboseBool(txhash0)
	if err != nil {
		return nil, fmt.Errorf("GetRawTransactionVerboseBool err: %s", err.Error())
	}

	txhash1, _ := chainhash.NewHashFromStr(txRawResult0.Vin[0].Txid)
	txRawResult1, err := e.node.GetRawTransactionVerboseBool(txhash1)
	if err != nil {
		return nil, fmt.Errorf("GetRawTransactionVerboseBool err: %s", err.Error())
	}

	if meme.Op == "deploy" {
		meme.HolderAddress = tx.Vout[0].ScriptPubKey.Addresses[0]
		meme.TickId = tx.Hash
		if tx.Vout[0].Value != 0.001 {
			return nil, fmt.Errorf("the amount of tokens exceeds the 0.0001")
		}

		if meme.HolderAddress != txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0] {
			return nil, fmt.Errorf("the address is not the same as the previous transaction")
		}
	}

	if meme.Op == "transfer" {
		meme.HolderAddress = txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0]
		meme.ToAddress = tx.Vout[0].ScriptPubKey.Addresses[0]
	}

	meme.FeeAddress = txRawResult0.Vout[tx.Vin[0].Vout].ScriptPubKey.Addresses[0]

	err = e.dbc.DB.Save(meme).Error
	if err != nil {
		return nil, fmt.Errorf("save err: %s", err.Error())
	}

	return meme, nil
}

func (e *Explorer) memeDeploy(meme *models.Meme20Info) error {

	tx := e.dbc.DB.Begin()
	meme20c := &models.Meme20Collect{}
	err := tx.Where("tick_id = ?", meme.TickId).First(meme20c).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		update := make(map[string]interface{})
		update["tick"] = meme.Tick
		update["name"] = meme.Name
		update["max_"] = meme.Max
		update["dec_"] = 8

		err = tx.Model(&models.Meme20Collect{}).Where("tick_id = ?", meme.TickId).Updates(update).Error
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("update err: %s", err.Error())
		}

	} else {
		meme20c = &models.Meme20Collect{
			Tick:          meme.Tick,
			TickId:        meme.TickId,
			Name:          meme.Name,
			Max:           meme.Max,
			Dec:           8,
			HolderAddress: meme.HolderAddress,
		}

		err := tx.Create(meme20c).Error
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("save err: %s", err.Error())
		}
	}

	meme20ca := &models.Meme20CollectAddress{
		TickId:        meme.TickId,
		Amt:           meme.Max,
		HolderAddress: meme.HolderAddress,
	}

	err = tx.Create(meme20ca).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("save err: %s", err.Error())
	}

	err = tx.Model(&models.Meme20Info{}).Where("tx_hash = ?", meme.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("update err: %s", err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("Commit err: %s", err.Error())
	}

	return nil
}

func (e *Explorer) meme20Transfer(meme20 *models.Meme20Info) error {

	tx := e.dbc.DB.Begin()
	err := e.dbc.TransferMeme20(tx, meme20.TickId, meme20.HolderAddress, meme20.ToAddress, meme20.Amt.Int(), meme20.TxHash, meme20.BlockNumber, false)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.Meme20Info{}).Where("tx_hash = ?", meme20.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Update err: %s", err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("Commit err: %s", err.Error())
	}

	return nil
}

func (e *Explorer) meme20Fork(tx *gorm.DB, height int64) error {

	log.Info("fork", "meme20", height)

	var meme20Reverts []*models.Meme20Revert
	err := tx.Model(&models.Meme20Revert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&meme20Reverts).Error

	if err != nil {
		return fmt.Errorf("meme20 revert error: %v", err)
	}

	for _, revert := range meme20Reverts {
		if revert.ToAddress != "" && revert.FromAddress == "" {
			err = e.dbc.BurnMeme20(tx, revert.TickId, revert.ToAddress, revert.Amt.Int(), "", 0, true)
			if err != nil {
				return fmt.Errorf("meme20 fork burn error: %v", err)
			}
		} else if revert.FromAddress != "" && revert.ToAddress == "" {
			err = e.dbc.MintMeme20(tx, revert.TickId, revert.FromAddress, revert.Amt.Int(), "", 0, true)
			if err != nil {
				return fmt.Errorf("meme20 fork mint error: %v", err)
			}
		} else {
			err = e.dbc.TransferMeme20(tx, revert.TickId, revert.ToAddress, revert.FromAddress, revert.Amt.Int(), "", 0, true)
			if err != nil {
				return fmt.Errorf("meme20 fork transfer error: %v", err)
			}
		}
	}

	return nil
}
