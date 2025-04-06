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
	"math/big"
)

const (
	PumpCreateFee = 500000000
	PumpTipFee    = 10000000
)

func (e *Explorer) pumpDecode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.PumpInfo, error) {

	err := e.dbc.DB.Where("tx_hash = ?", tx.Hash).First(&models.PumpInfo{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("pump already exist or err %s", tx.Hash)
	}

	dogeDepositAmt := big.NewInt(0)

	param := &models.PumpInscription{}
	err = json.Unmarshal(pushedData, param)
	if err != nil {
		return nil, fmt.Errorf("json Unmarshal err: %s", err.Error())
	}

	pump, err := utils.ConvertPump(param)
	if err != nil {
		return nil, fmt.Errorf("ConvertPump err: %s", err.Error())
	}

	if pump.Doge == 1 {
		if pump.Op == "deploy" {
			if pump.Tick0Id == "WDOGE(WRAPPED-DOGE)" {
				dogeDepositAmt.Add(dogeDepositAmt, pump.Amt0.Int())
			}

			if pump.Tick1Id == "WDOGE(WRAPPED-DOGE)" {
				dogeDepositAmt.Add(dogeDepositAmt, pump.Amt1.Int())
			}
		}

		if pump.Op == "trade" {
			if pump.Tick0Id == "WDOGE(WRAPPED-DOGE)" {
				dogeDepositAmt.Add(dogeDepositAmt, pump.Amt0.Int())
			}
		}
	}

	pump.OrderId = uuid.New().String()
	pump.FeeTxHash = tx.Vin[0].Txid
	pump.TxHash = tx.Hash
	pump.BlockHash = tx.BlockHash
	pump.BlockNumber = number
	pump.BlockTime = tx.Blocktime
	pump.HolderAddress = tx.Vout[0].ScriptPubKey.Addresses[0]
	pump.OrderStatus = 1

	if pump.Op == "deploy" {
		pump.Tick0Id = pump.TxHash
		if dogeDepositAmt.Cmp(big.NewInt(0)) > 0 {
			if len(tx.Vout) < 5 {
				return nil, fmt.Errorf("deposit op error, vout length is not 5")
			}

			if utils.Float64ToBigInt(tx.Vout[3].Value*100000000).Cmp(big.NewInt(PumpCreateFee)) < 0 {
				return nil, fmt.Errorf("the amount of tokens is incorrect %f %s", tx.Vout[1].Value, utils.Float64ToBigInt(tx.Vout[3].Value*100000000).String())
			}

			if tx.Vout[3].ScriptPubKey.Addresses[0] != pumpCreateFeeAddress {
				return nil, fmt.Errorf("the address is incorrect")
			}

			if utils.Float64ToBigInt(tx.Vout[4].Value*100000000).Cmp(big.NewInt(PumpTipFee)) < 0 {
				return nil, fmt.Errorf("the amount of tokens is incorrect fee %f", tx.Vout[4].Value)
			}

			if tx.Vout[4].ScriptPubKey.Addresses[0] != pumpTipAddress {
				return nil, fmt.Errorf("the address is incorrect")
			}

		} else {
			if len(tx.Vout) < 3 {
				return nil, fmt.Errorf("deposit op error, vout length is not 5")
			}

			if utils.Float64ToBigInt(tx.Vout[1].Value*100000000).Cmp(big.NewInt(PumpCreateFee)) < 0 {
				return nil, fmt.Errorf("the amount of tokens is incorrect %f %s", tx.Vout[1].Value, utils.Float64ToBigInt(tx.Vout[1].Value*100000000).String())
			}

			if tx.Vout[1].ScriptPubKey.Addresses[0] != pumpCreateFeeAddress {
				return nil, fmt.Errorf("the address is incorrect")
			}

			if utils.Float64ToBigInt(tx.Vout[2].Value*100000000).Cmp(big.NewInt(PumpTipFee)) < 0 {
				return nil, fmt.Errorf("the amount of tokens is incorrect fee %f", tx.Vout[2].Value)
			}

			if tx.Vout[2].ScriptPubKey.Addresses[0] != pumpTipAddress {
				return nil, fmt.Errorf("the address is incorrect")
			}
		}
	}

	if pump.Op == "trade" {

		if dogeDepositAmt.Cmp(big.NewInt(0)) > 0 {
			if len(tx.Vout) < 4 {
				return nil, fmt.Errorf("deposit op error, vout length is not 5")
			}

			if utils.Float64ToBigInt(tx.Vout[3].Value*100000000).Cmp(big.NewInt(PumpTipFee)) < 0 {
				return nil, fmt.Errorf("the amount of tokens is incorrect fee %f", tx.Vout[4].Value)
			}

			if tx.Vout[3].ScriptPubKey.Addresses[0] != pumpTipAddress {
				return nil, fmt.Errorf("the address is incorrect")
			}

		} else {
			if len(tx.Vout) < 2 {
				return nil, fmt.Errorf("deposit op error, vout length is not 5")
			}

			if utils.Float64ToBigInt(tx.Vout[1].Value*100000000).Cmp(big.NewInt(PumpTipFee)) < 0 {
				return nil, fmt.Errorf("the amount of tokens is incorrect fee %f", tx.Vout[2].Value)
			}

			if tx.Vout[1].ScriptPubKey.Addresses[0] != pumpTipAddress {
				return nil, fmt.Errorf("the address is incorrect")
			}
		}
	}

	txhash0, _ := chainhash.NewHashFromStr(pump.FeeTxHash)
	txRawResult0, err := e.node.GetRawTransactionVerboseBool(txhash0)
	if err != nil {
		return nil, CHAIN_NETWORK_ERR
	}

	pump.FeeAddress = txRawResult0.Vout[pump.FeeTxIndex].ScriptPubKey.Addresses[0]

	txhash1, _ := chainhash.NewHashFromStr(txRawResult0.Vin[0].Txid)
	txRawResult1, err := e.node.GetRawTransactionVerboseBool(txhash1)
	if err != nil {
		return nil, CHAIN_NETWORK_ERR
	}

	if pump.HolderAddress != txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0] {
		return nil, fmt.Errorf("the address is not the same as the previous transaction")
	}

	err = e.dbc.DB.Create(pump).Error
	if err != nil {
		return nil, fmt.Errorf("pump create err: %s", err.Error())
	}

	if dogeDepositAmt.Cmp(big.NewInt(0)) > 0 {
		if len(tx.Vout) < 3 {
			return nil, fmt.Errorf("the number of outputs is incorrect")
		}

		fee := big.NewInt(0)
		fee.Mul(dogeDepositAmt, big.NewInt(3))
		fee.Div(fee, big.NewInt(1000))
		if fee.Cmp(big.NewInt(50000000)) == -1 {
			fee = big.NewInt(50000000)
		}

		if utils.Float64ToBigInt(tx.Vout[1].Value*100000000).Cmp(dogeDepositAmt) < 0 {
			return nil, fmt.Errorf("the amount of tokens is incorrect %f %s", tx.Vout[1].Value, utils.Float64ToBigInt(tx.Vout[1].Value*100000000).String())
		}

		if tx.Vout[1].ScriptPubKey.Addresses[0] != wdogeCoolAddress {
			return nil, fmt.Errorf("the address is incorrect")
		}

		if utils.Float64ToBigInt(tx.Vout[2].Value*100000000).Cmp(fee) < 0 {
			return nil, fmt.Errorf("the amount of tokens is incorrect fee %f", tx.Vout[2].Value)
		}

		if tx.Vout[2].ScriptPubKey.Addresses[0] != wdogeFeeAddress {
			return nil, fmt.Errorf("the address is incorrect")
		}
	}

	return pump, nil
}

func (e *Explorer) pumpDeploy(db *gorm.DB, pump *models.PumpInfo) error {

	log.Info("explorer", "p", "pump", "op", "deploy", "tx_hash", pump.TxHash)

	err := e.dbc.PumpDeploy(db, pump)
	if err != nil {
		return fmt.Errorf("pumpDeploy Create err: %s", err.Error())
	}

	update := map[string]interface{}{
		"order_status": 0,
		"amt0":         pump.Amt1.String(),
		"amt0_out":     pump.Amt0Out.String(),
		"amt1_out":     pump.Amt1Out.String(),
	}
	err = db.Model(&models.PumpInfo{}).Where("tx_hash = ? and tx_index = ?", pump.TxHash, pump.TxIndex).Updates(update).Error
	if err != nil {
		return fmt.Errorf("pumpDeploy Update err: %s", err.Error())
	}

	return nil
}

func (e *Explorer) pumpTrade(db *gorm.DB, pump *models.PumpInfo) error {

	log.Info("explorer", "p", "pump", "op", "trade", "tx_hash", pump.TxHash)

	err := e.dbc.PumpTrade(db, pump)
	if err != nil {
		return fmt.Errorf("PumpTrade error: %v", err)
	}

	updates := map[string]interface{}{
		"order_status": 0,
		"amt1_out":     pump.Amt1Out.String(),
		"tick1_id":     pump.Tick1Id,
		"tick0":        pump.Tick0,
		"tick1":        pump.Tick1,
	}

	err = db.Model(&models.PumpInfo{}).Where("tx_hash = ? and tx_index = ?", pump.TxHash, pump.TxIndex).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("PumpTrade Update err: %s", err.Error())
	}

	return nil
}

func (e *Explorer) pumpFork(tx *gorm.DB, height int64) error {

	log.Info("fork", "pump", height)
	var pumpReverts []*models.PumpRevert
	err := tx.Model(&models.PumpRevert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&pumpReverts).Error

	if err != nil {
		return fmt.Errorf("PumpRevert error: %v", err)
	}

	for _, revert := range pumpReverts {
		if revert.Op == "deploy" {

			err = tx.Where("tick0_id = ?", revert.TickId).Delete(&models.PumpLiquidity{}).Error
			if err != nil {
				return fmt.Errorf("PumpLiquidity error: %v", err)
			}

			err = tx.Where("tick_id = ?", revert.TickId).Delete(&models.Meme20Collect{}).Error
			if err != nil {
				return fmt.Errorf("Meme20Collect error: %v", err)
			}

			err = tx.Where("tick_id = ?", revert.TickId).Delete(&models.Meme20CollectAddress{}).Error
			if err != nil {
				return fmt.Errorf("Meme20CollectAddress error: %v", err)
			}
		}

		if revert.Op == "trade" {
			err = tx.Model(&models.PumpLiquidity{}).Where("tick0_id = ?", revert.TickId).Updates(map[string]interface{}{
				"amt0": revert.Amt0,
				"amt1": revert.Amt1,
			}).Error
			if err != nil {
				return fmt.Errorf("PumpLiquidity error: %v", err)
			}
		}

		//if revert.Op == "finish" {
		//}
	}

	var pumpInviteReverts []*models.PumpInviteRewardRevert
	err = tx.Model(&models.PumpInviteRewardRevert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&pumpInviteReverts).Error

	if err != nil {
		return fmt.Errorf("PumpRevert error: %v", err)
	}

	for _, revert := range pumpInviteReverts {
		err = tx.Model(&models.PumpInviteReward{}).Where("holder_address = ? and invite_address = ?", revert.HolderAddress, revert.InviteAddress).Updates(map[string]interface{}{
			"invite_reward": revert.InviteReward,
		}).Error
		if err != nil {
			return fmt.Errorf("PumpInviteReward error: %v", err)
		}
	}

	return nil
}
