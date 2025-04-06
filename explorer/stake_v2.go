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

func (e *Explorer) stakeV2Decode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.StakeV2Info, error) {

	err := e.dbc.DB.Where("tx_hash = ?", tx.Txid).First(&models.StakeV2Info{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("stake already exist or err %s", tx.Txid)
	}

	param := &models.StakeV2Inscription{}
	err = json.Unmarshal(pushedData, param)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	stake, err := utils.ConvertStakeV2(param)
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

	if stake.Op == "create" {
		stake.StakeId = tx.Txid
	}

	if stake.Op == "stake" {
		stakec := &models.StakeV2Collect{}
		err := e.dbc.DB.Where("stake_id = ?", stake.StakeId).First(stakec).Error
		if err != nil {
			return nil, fmt.Errorf("stake id not found")
		}

		stake.Tick0 = stakec.Tick0
		stake.Tick1 = stakec.Tick1
		stake.Reward = stakec.Reward
		stake.EachReward = stakec.EachReward
	}

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
		return nil, fmt.Errorf("SaveStakeV2 err: %s", err.Error())
	}

	return stake, nil
}

func (e *Explorer) stakeV2Create(stake *models.StakeV2Info) error {
	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(stake.StakeId+"--STAKE-V2"), &chaincfg.MainNetParams)

	tx := e.dbc.DB.Begin()
	err := e.dbc.StakeV2Create(tx, stake, reservesAddress.String())
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.StakeV2Info{}).Where("tx_hash = ?", stake.TxHash).Update("order_status", 0).Error
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

func (e *Explorer) stakeV2Stake(stake *models.StakeV2Info) error {

	tx := e.dbc.DB.Begin()
	err := e.dbc.StakeV2Stake(tx, stake)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.StakeV2Info{}).Where("tx_hash = ?", stake.TxHash).Update("order_status", 0).Error
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

func (e *Explorer) stakeV2UnStake(stake *models.StakeV2Info) error {
	tx := e.dbc.DB.Begin()
	err := e.dbc.StakeV2UnStake(tx, stake)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.StakeV2Info{}).Where("tx_hash = ?", stake.TxHash).Update("order_status", 0).Error
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

func (e *Explorer) stakeV2GetReward(stake *models.StakeV2Info) error {
	tx := e.dbc.DB.Begin()
	err := e.dbc.StakeV2GetReward(tx, stake)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.StakeV2Info{}).Where("tx_hash = ?", stake.TxHash).Update("order_status", 0).Error
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

func (e *Explorer) stakeV2Fork(tx *gorm.DB, height int64) error {

	log.Info("fork", "stake_v2", height)
	//stake_v2
	var stakeV2Reverts []*models.StakeV2Revert
	err := tx.Model(&models.StakeV2Revert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&stakeV2Reverts).Error
	if err != nil {
		return fmt.Errorf("FindStakeV2Revert error: %v", err)
	}

	for _, revert := range stakeV2Reverts {
		if revert.Op == "create" {

			err = tx.Where("stake_id = ?", revert.StakeId).Delete(&models.StakeV2Collect{}).Error
			if err != nil {
				return fmt.Errorf("StakeV2Collect error: %v", err)
			}

			err = tx.Where("tick = ?", revert.Tick).Delete(&models.Drc20Collect{}).Error
			if err != nil {
				return fmt.Errorf("Drc20Collect error: %v", err)
			}
		}

		if revert.Op == "stake-pool" {

			err = tx.Model(&models.StakeV2Collect{}).Where("stake_id = ?", revert.StakeId).Updates(map[string]interface{}{
				"total_staked":         revert.Amt,
				"acc_reward_per_share": revert.AccRewardPerShare,
				"last_block":           revert.LastBlock,
			}).Error
			if err != nil {
				return fmt.Errorf("StakeV2Collect error: %v", err)
			}
		}

		if revert.Op == "stake-create" {
			err = tx.Where("stake_id = ? AND holder_address = ? ", revert.StakeId, revert.HolderAddress).Delete(&models.StakeV2CollectAddress{}).Error
			if err != nil {
				return fmt.Errorf("StakeV2CollectAddress error: %v", err)
			}
		}

		if revert.Op == "stake" {

			err = tx.Model(&models.StakeV2CollectAddress{}).Where("stake_id = ? AND holder_address = ?", revert.StakeId, revert.HolderAddress).Updates(map[string]interface{}{
				"amt":            revert.Amt,
				"reward_debt":    revert.RewardDebt,
				"pending_reward": revert.PendingReward,
			}).Error
			if err != nil {
				return fmt.Errorf("StakeV2CollectAddress error: %v", err)
			}
		}

		if revert.Op == "unstake" {

			err = tx.Model(&models.StakeV2CollectAddress{}).Where("stake_id = ? AND holder_address = ?", revert.StakeId, revert.HolderAddress).Updates(map[string]interface{}{
				"amt":            revert.Amt,
				"reward_debt":    revert.RewardDebt,
				"pending_reward": revert.PendingReward,
			}).Error
			if err != nil {
				return fmt.Errorf("StakeV2CollectAddress error: %v", err)
			}
		}

		if revert.Op == "getreward" {

			err = tx.Model(&models.StakeV2CollectAddress{}).Where("stake_id = ? AND holder_address = ?", revert.StakeId, revert.HolderAddress).Updates(map[string]interface{}{
				"reward_debt":    revert.RewardDebt,
				"pending_reward": revert.PendingReward,
			}).Error
			if err != nil {
				return fmt.Errorf("StakeV2CollectAddress error: %v", err)
			}
		}
	}

	return nil
}
