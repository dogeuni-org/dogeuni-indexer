package explorer

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/btcutil"
	"github.com/dogecoinw/doged/chaincfg"
	"github.com/dogecoinw/doged/chaincfg/chainhash"
	"github.com/dogecoinw/go-dogecoin/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func (e *Explorer) fileExchangeDecode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.FileExchangeInfo, error) {

	err := e.dbc.DB.Where("tx_hash = ?", tx.Hash).First(&models.FileExchangeInfo{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("file-exchange already exist or err %s", tx.Hash)
	}

	param := &models.FileExchangeInscription{}
	err = json.Unmarshal(pushedData, param)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	ex, err := utils.ConvertFileExchange(param)
	if err != nil {
		return nil, fmt.Errorf("exchange err: %s", err.Error())
	}

	ex.OrderId = uuid.New().String()
	ex.FeeTxHash = tx.Vin[0].Txid
	ex.TxHash = tx.Hash
	ex.BlockHash = tx.BlockHash
	ex.BlockNumber = number
	ex.OrderStatus = 1
	ex.UpdateDate = models.LocalTime(time.Now().Unix())
	ex.CreateDate = models.LocalTime(time.Now().Unix())

	ex.HolderAddress = tx.Vout[0].ScriptPubKey.Addresses[0]
	if ex.Op == "create" {
		ex.ExId = tx.Hash
	}

	if ex.Op == "trade" {

		exc := &models.FileExchangeCollect{}
		err := e.dbc.DB.Where("ex_id = ? ", ex.ExId).First(exc).Error
		if err != nil {
			return nil, fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		ex.FileId = exc.FileId
	}

	if ex.Op == "cancel" {
		exc := &models.FileExchangeCollect{}
		err := e.dbc.DB.Where("ex_id = ? ", ex.ExId).First(exc).Error
		if err != nil {
			return nil, fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		ex.FileId = exc.FileId
		ex.Tick = exc.Tick
		ex.Amt = exc.Amt

	}

	txhash0, _ := chainhash.NewHashFromStr(tx.Vin[0].Txid)
	txRawResult0, err := e.node.GetRawTransactionVerboseBool(txhash0)
	if err != nil {
		return nil, CHAIN_NETWORK_ERR
	}

	ex.FeeAddress = txRawResult0.Vout[tx.Vin[0].Vout].ScriptPubKey.Addresses[0]

	txhash1, _ := chainhash.NewHashFromStr(txRawResult0.Vin[0].Txid)
	txRawResult1, err := e.node.GetRawTransactionVerboseBool(txhash1)
	if err != nil {
		return nil, CHAIN_NETWORK_ERR
	}

	if ex.HolderAddress != txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0] {
		return nil, fmt.Errorf("the address is not the same as the previous transaction")
	}

	err = e.dbc.DB.Create(ex).Error
	if err != nil {
		return nil, fmt.Errorf("InstallFileExchangeInfo err: %s", err.Error())
	}

	return ex, nil
}

func (e *Explorer) fileExchangeCreate(ex *models.FileExchangeInfo) error {
	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(ex.ExId), &chaincfg.MainNetParams)
	tx := e.dbc.DB.Begin()

	err := e.dbc.FileExchangeCreate(tx, ex, reservesAddress.String())
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.FileExchangeInfo{}).Where("tx_hash = ?", ex.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (e *Explorer) fileExchangeTrade(ex *models.FileExchangeInfo) error {
	tx := e.dbc.DB.Begin()

	err := e.dbc.FileExchangeTrade(tx, ex)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.FileExchangeInfo{}).Where("tx_hash = ?", ex.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func (e *Explorer) fileExchangeCancel(ex *models.FileExchangeInfo) error {
	tx := e.dbc.DB.Begin()
	err := e.dbc.FileExchangeCancel(tx, ex)
	if err != nil {
		tx.Rollback()
		return nil
	}

	err = tx.Model(&models.FileExchangeInfo{}).Where("tx_hash = ?", ex.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return nil
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return nil
	}
	return nil
}

func (e *Explorer) fileExchangeFork(tx *gorm.DB, height int64) error {

	log.Info("fork", "file_exchange", height)
	// FileExchange
	var fileExchangeReverts []*models.FileExchangeRevert
	err := tx.Model(&models.FileExchangeRevert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&fileExchangeReverts).Error

	if err != nil {
		return fmt.Errorf("exchangeFork error: %v", err)
	}

	for _, revert := range fileExchangeReverts {
		if revert.Op == "create" {
			err = tx.Where("ex_id = ?", revert.ExId).Delete(&models.FileExchangeCollect{}).Error
			if err != nil {
				return fmt.Errorf("delete error: %v", err)
			}
		}

		if revert.Op == "trade" {

			ec := &models.FileExchangeCollect{}
			err = tx.Where("ex_id = ?", revert.ExId).First(ec).Error
			if err != nil {
				return fmt.Errorf("error: %v", err)
			}

			err = tx.Model(&models.ExchangeCollect{}).
				Where("ex_id = ?", revert.ExId).
				Updates(map[string]interface{}{
					"amt_finish": models.NewNumber(0),
				}).Error

			if err != nil {
				return fmt.Errorf("update exchange_collect error: %v", err)
			}
		}

		if revert.Op == "cancel" {

			ec := &models.ExchangeCollect{}
			err = tx.Where("ex_id = ?", revert.ExId).First(ec).Error
			if err != nil {
				return fmt.Errorf("select exchange_collect error: %v", err)
			}

			err = tx.Model(&models.ExchangeCollect{}).Where("ex_id = ?", revert.ExId).Update("amt_finish", models.NewNumber(0)).Error
			if err != nil {
				return fmt.Errorf("update exchange_collect error: %v", err)
			}
		}
	}

	return nil

}
