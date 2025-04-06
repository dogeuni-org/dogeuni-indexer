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

func (e *Explorer) crossDecode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.CrossInfo, error) {

	err := e.dbc.DB.Where("tx_hash = ?", tx.Txid).First(&models.CrossInfo{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("cross already exist or err %s", tx.Txid)
	}

	param := &models.CrossInscription{}
	err = json.Unmarshal(pushedData, param)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	cross, err := utils.ConvertCross(param)
	if err != nil {
		return nil, fmt.Errorf("ConvertCross err: %s", err.Error())
	}

	cross.OrderId = uuid.New().String()
	cross.FeeTxHash = tx.Vin[0].Txid
	cross.TxHash = tx.Hash
	cross.BlockHash = tx.BlockHash
	cross.BlockNumber = number
	cross.OrderStatus = 1
	cross.HolderAddress = tx.Vout[0].ScriptPubKey.Addresses[0]

	txhash0, _ := chainhash.NewHashFromStr(tx.Vin[0].Txid)
	txRawResult0, err := e.node.GetRawTransactionVerboseBool(txhash0)
	if err != nil {
		return nil, CHAIN_NETWORK_ERR
	}

	txhash1, _ := chainhash.NewHashFromStr(txRawResult0.Vin[0].Txid)
	txRawResult1, err := e.node.GetRawTransactionVerboseBool(txhash1)
	if err != nil {
		return nil, CHAIN_NETWORK_ERR
	}

	if cross.Op == "mint" {

		txhash1, _ := chainhash.NewHashFromStr(txRawResult0.Vin[0].Txid)
		txRawResult1, err := e.node.GetRawTransactionVerboseBool(txhash1)
		if err != nil {
			return nil, fmt.Errorf("GetRawTransactionVerboseBool err: %s", err.Error())
		}

		cross.HolderAddress = txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0]

	} else {
		if cross.HolderAddress != txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0] {
			return nil, fmt.Errorf("the address is not the same as the previous transaction")
		}

	}

	err = e.dbc.DB.Create(cross).Error
	if err != nil {
		return nil, fmt.Errorf("err: %s", err.Error())
	}

	return cross, nil
}

func (e *Explorer) crossDeploy(cross *models.CrossInfo) error {

	tx := e.dbc.DB.Begin()
	err := e.dbc.CrossDeploy(tx, cross)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.CrossInfo{}).Where("tx_hash = ?", cross.TxHash).Update("order_status", 0).Error
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

func (e *Explorer) crossMint(cross *models.CrossInfo) error {

	tx := e.dbc.DB.Begin()
	err := e.dbc.CrossMint(tx, cross)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.CrossInfo{}).Where("tx_hash = ?", cross.TxHash).Update("order_status", 0).Error
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

func (e *Explorer) crossBurn(cross *models.CrossInfo) error {

	tx := e.dbc.DB.Begin()
	err := e.dbc.CrossBurn(tx, cross)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.CrossInfo{}).Where("tx_hash = ?", cross.TxHash).Update("order_status", 0).Error
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

func (e *Explorer) crossFork(tx *gorm.DB, height int64) error {
	log.Info("fork", "cross", height)
	//cross
	var crossReverts []*models.CrossRevert
	err := tx.Model(&models.CrossRevert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&crossReverts).Error
	if err != nil {
		return fmt.Errorf("FindCrossRevert error: %v", err)
	}

	for _, revert := range crossReverts {
		if revert.Op == "deploy" {

			err = tx.Where("tick = ?", revert.Tick).Delete(&models.CrossCollect{}).Error
			if err != nil {
				return fmt.Errorf("CrossCollect error: %v", err)
			}

			err = tx.Where("tick = ?", revert.Tick).Delete(&models.Drc20Collect{}).Error
			if err != nil {
				return fmt.Errorf("Drc20Collect error: %v", err)
			}
		}
	}

	return nil
}
