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
)

func (e *Explorer) boxDecode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.BoxInfo, error) {

	err := e.dbc.DB.Where("tx_hash = ?", tx.Hash).First(&models.BoxInfo{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("box already exist or err %s", tx.Hash)
	}

	param := &models.BoxInscription{}
	err = json.Unmarshal(pushedData, param)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	box, err := utils.ConvertBox(param)
	if err != nil {
		return nil, fmt.Errorf("ConvertBox err: %s", err.Error())
	}

	box.OrderId = uuid.New().String()
	box.FeeTxHash = tx.Vin[0].Txid
	box.TxHash = tx.Hash
	box.BlockHash = tx.BlockHash
	box.BlockNumber = number
	box.OrderStatus = 1

	if len(tx.Vout) < 1 {
		return nil, fmt.Errorf("vout length is not enough")
	}

	box.HolderAddress = tx.Vout[0].ScriptPubKey.Addresses[0]

	txHashIn, _ := chainhash.NewHashFromStr(tx.Vin[0].Txid)
	txRawResult0, err := e.node.GetRawTransactionVerboseBool(txHashIn)
	if err != nil {
		return nil, CHAIN_NETWORK_ERR
	}

	box.FeeAddress = txRawResult0.Vout[tx.Vin[0].Vout].ScriptPubKey.Addresses[0]

	txHashIn1, _ := chainhash.NewHashFromStr(txRawResult0.Vin[0].Txid)
	txRawResult1, err := e.node.GetRawTransactionVerboseBool(txHashIn1)
	if err != nil {
		return nil, CHAIN_NETWORK_ERR
	}

	if box.HolderAddress != txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0] {
		return nil, fmt.Errorf("the address is not the same as the previous transaction")
	}

	err = e.dbc.DB.Save(box).Error
	if err != nil {
		return nil, err
	}

	return box, nil
}

func (e *Explorer) boxDeploy(box *models.BoxInfo) error {

	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(box.Tick0+"--BOX"), &chaincfg.MainNetParams)

	tx := e.dbc.DB.Begin()
	err := e.dbc.BoxDeploy(tx, box, reservesAddress.String())
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.BoxInfo{}).Where("tx_hash = ?", box.TxHash).Update("order_status", 0).Error
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

func (e *Explorer) boxMint(box *models.BoxInfo) error {

	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(box.Tick0+"--BOX"), &chaincfg.MainNetParams)
	tx := e.dbc.DB.Begin()
	err := e.dbc.BoxMint(tx, box, reservesAddress.String())
	if err != nil {
		tx.Rollback()
		return err
	}

	updates := map[string]interface{}{
		"tick1":        box.Tick1,
		"max_":         box.Max,
		"amt0":         box.Amt0,
		"order_status": 0,
	}

	err = tx.Model(&models.BoxInfo{}).Where("tx_hash = ?", box.TxHash).Updates(updates).Error
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

func (e *Explorer) boxFork(tx *gorm.DB, height int64) error {

	log.Info("fork", "box", height)

	// box
	err := tx.Exec("update box_collect a, drc20_collect_address b set a.liqamt_finish = b.amt_sum where a.tick1 = b.tick and a.reserves_address = b.holder_address").Error
	if err != nil {
		return fmt.Errorf("update box_collect error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.BoxCollectAddress{}).Error
	if err != nil {
		return fmt.Errorf("DeleteBoxCollectAddress error: %v", err)
	}

	var boxReverts []*models.BoxRevert
	err = tx.Model(&models.BoxRevert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&boxReverts).Error

	if err != nil {
		return fmt.Errorf("box revert error: %v", err)
	}

	for _, revert := range boxReverts {
		if revert.Op == "deploy" {

			err = tx.Where("tick = ?", revert.Tick0).Delete(&models.Drc20Collect{}).Error
			if err != nil {
				return fmt.Errorf("delete drc20_collect error: %v", err)
			}

			err = tx.Where("tick0 = ?", revert.Tick0).Delete(&models.BoxCollect{}).Error
			if err != nil {
				return fmt.Errorf("delete box_collect error: %v", err)
			}
		}

		if revert.Op == "finish" {
			err = tx.Where("tick0 = ? and tick1 = ?", revert.Tick0, revert.Tick1).Delete(&models.SwapLiquidity{}).Error
			if err != nil {
				return fmt.Errorf("delete swap_info error: %v", err)
			}
		}

		if revert.Op == "refund-drc20" {

			drc20c := &models.Drc20Collect{
				Tick:          revert.Tick0,
				Max:           revert.Max,
				Dec:           8,
				HolderAddress: revert.HolderAddress,
				TxHash:        revert.TxHash,
			}

			err := tx.Create(drc20c).Error
			if err != nil {
				return fmt.Errorf("create drc20_collect error: %v", err)
			}
		}
	}

	err = tx.Model(&models.BoxCollect{}).
		Where("liqblock > ?", height).
		Update("is_del", 0).Error
	if err != nil {
		return fmt.Errorf("update box_collect error: %v", err)
	}

	return nil

}
