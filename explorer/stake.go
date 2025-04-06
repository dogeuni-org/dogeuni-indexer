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
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (e Explorer) stakeDecode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.StakeInfo, error) {

	err := e.dbc.DB.Where("tx_hash = ?", tx.Txid).First(&models.StakeInfo{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("stake already exist or err %s", tx.Txid)
	}

	param := &models.StakeInscription{}
	err = json.Unmarshal(pushedData, param)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	stake, err := utils.ConvertStake(param)
	if err != nil {
		return nil, fmt.Errorf("ConvertWDoge err: %s", err.Error())
	}

	if len(tx.Vout) < 1 {
		return nil, fmt.Errorf("op error, vout length is not 0")
	}

	stake.OrderId = uuid.New().String()
	stake.FeeTxHash = tx.Vin[0].Txid
	stake.TxHash = tx.Txid
	stake.BlockHash = tx.BlockHash
	stake.BlockNumber = number
	stake.OrderStatus = 1

	stake.HolderAddress = tx.Vout[0].ScriptPubKey.Addresses[0]

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

	if stake.HolderAddress != txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0] {
		return nil, fmt.Errorf("The address is not the same as the previous transaction")
	}

	err = e.dbc.DB.Save(stake).Error
	if err != nil {
		return nil, fmt.Errorf("SaveStake err: %s", err.Error())
	}

	return stake, nil
}

func (e Explorer) stakeStake(stake *models.StakeInfo) error {
	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(stake.Tick+"--STAKE"), &chaincfg.MainNetParams)

	tx := e.dbc.DB.Begin()
	err := e.dbc.StakeStake(tx, stake, reservesAddress.String())
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.StakeInfo{}).Where("tx_hash = ?", stake.TxHash).Update("order_status", 0).Error
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

func (e Explorer) stakeUnStake(stake *models.StakeInfo) error {

	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(stake.Tick+"--STAKE"), &chaincfg.MainNetParams)

	tx := e.dbc.DB.Begin()
	err := e.dbc.StakeUnStake(tx, stake, reservesAddress.String())
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.StakeInfo{}).Where("tx_hash = ?", stake.TxHash).Update("order_status", 0).Error
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

func (e Explorer) stakeGetAllReward(stake *models.StakeInfo) error {

	tx := e.dbc.DB.Begin()
	err := e.dbc.StakeGetReward(tx, stake)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.StakeInfo{}).Where("tx_hash = ?", stake.TxHash).Update("order_status", 0).Error
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
