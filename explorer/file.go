package explorer

import (
	"bytes"
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
	"strings"
	"time"
)

func (e *Explorer) fileDecode(tx *btcjson.TxRawResult, number int64) (*models.FileInfo, error) {

	err := e.dbc.DB.Where("tx_hash = ?", tx.Hash).First(&models.FileInfo{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("file already exist or err %s", tx.Hash)
	}

	inscription, err := e.reDecodeFile(tx)
	if err != nil {
		return nil, fmt.Errorf("reDecodeFile err: %s", err.Error())
	}

	file, err := utils.ConvertFile(inscription)
	if err != nil {
		return nil, fmt.Errorf("convertNft err: %s", err.Error())
	}

	file.OrderId = uuid.New().String()
	file.FeeTxHash = tx.Vin[0].Txid

	file.TxHash = tx.Hash
	file.BlockHash = tx.BlockHash
	file.BlockNumber = number
	file.OrderStatus = 1
	file.UpdateDate = models.LocalTime(time.Now().Unix())
	file.CreateDate = models.LocalTime(time.Now().Unix())

	if file.Op == "deploy" {
		file.FileId = tx.Hash
		file.HolderAddress = tx.Vout[0].ScriptPubKey.Addresses[0]
		if tx.Vout[0].Value != 0.001 {
			return nil, fmt.Errorf("the amount of tokens exceeds the 0.0001")
		}
	}

	txHash0, _ := chainhash.NewHashFromStr(tx.Vin[0].Txid)
	txRawResult0, err := e.node.GetRawTransactionVerboseBool(txHash0)
	if err != nil {
		return nil, fmt.Errorf("getRawTransactionVerboseBool err: %s", err.Error())
	}

	if file.Op == "transfer" {

		txhash1, _ := chainhash.NewHashFromStr(txRawResult0.Vin[0].Txid)
		txRawResult1, err := e.node.GetRawTransactionVerboseBool(txhash1)
		if err != nil {
			return nil, fmt.Errorf("getRawTransactionVerboseBool err: %s", err.Error())
		}

		file.HolderAddress = txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0]
		file.ToAddress = tx.Vout[0].ScriptPubKey.Addresses[0]

		if file.HolderAddress == file.ToAddress {
			return nil, errors.New("the address is the same")
		}
	}

	// Unified entry: detect cardity JSON body and route to executeCardity
	if len(file.FileData) > 0 {
		var mp map[string]interface{}
		if json.Unmarshal(file.FileData, &mp) == nil {
			if p, ok := mp["p"].(string); ok {
				lp := strings.ToLower(p)
				if lp == "cardity" || lp == "cardinals" || lp == "cpl" {
					raw := string(file.FileData)
					if err := e.executeCardity(tx.Txid, tx.BlockHash, number, raw); err != nil {
						log.Error("file", "executeCardity", err, "tx_hash", tx.Txid)
					}
				}
			}
		}
	}

	reader := bytes.NewReader(file.FileData)

	hash, _ := e.ipfs.Add(reader)
	file.FilePath = "https://ipfs.unielon.com/ipfs/" + hash
	file.File = ""
	file.FileLength = len(file.FileData)
	file.FileType = "file"

	err = e.dbc.DB.Create(file).Error
	if err != nil {
		return nil, fmt.Errorf("createFileInfo err: %s", err.Error())
	}

	return file, nil
}

func (e *Explorer) fileDeploy(model *models.FileInfo) error {

	log.Info("explorer", "p", "file", "op", "deploy", "tx_hash", model.TxHash)

	tx := e.dbc.DB.Begin()
	err := e.dbc.FileDeploy(tx, model)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("deploy err: %s order_id: %s", err, model.OrderId)
	}

	err = tx.Model(&models.FileInfo{}).Where("tx_hash = ?", model.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("fileDeploy update status err: %s order_id: %s", err, model.OrderId)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("fileDeploy commit err: %s order_id: %s", err, model.OrderId)
	}

	return nil
}

func (e *Explorer) fileTransfer(model *models.FileInfo) error {

	log.Info("explorer", "p", "file", "op", "transfer", "tx_hash", model.TxHash)

	tx := e.dbc.DB.Begin()

	err := e.dbc.FileTransfer(tx, model)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("transfer err: %s order_id: %s", err, model.OrderId)
	}

	err = tx.Model(&models.FileInfo{}).Where("tx_hash = ?", model.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("fileTransfer update status err: %s order_id: %s", err, model.OrderId)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("fileTransfer commit err: %s order_id: %s", err, model.OrderId)
	}

	return nil
}

func (e *Explorer) fileFork(tx *gorm.DB, height int64) error {
	log.Info("fork", "file", height)
	// file
	var fileReverts []*models.FileRevert
	err := tx.Model(&models.FileRevert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&fileReverts).Error

	if err != nil {
		return fmt.Errorf("file revert error: %v", err)
	}

	for _, revert := range fileReverts {
		if revert.ToAddress != "" && revert.FromAddress == "" {
			err := tx.Where("file_id = ? AND holder_address = ?", revert.FileId, revert.ToAddress).
				Delete(&models.FileCollectAddress{}).Error
			if err != nil {
				return fmt.Errorf("fileFork burn error: %v", err)
			}
		} else {
			err = e.dbc.TransferFile(tx, revert.ToAddress, revert.FromAddress, revert.FileId, "", height, true)
			if err != nil {
				return fmt.Errorf("fileFork Transfer error: %v", err)
			}
		}
	}

	return nil
}
