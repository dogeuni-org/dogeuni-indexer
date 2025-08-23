package explorer

import (
	"encoding/json"
	"errors"
	"fmt"

	"dogeuni-indexer/models"
	"dogeuni-indexer/utils"

	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/btcutil"
	"github.com/dogecoinw/doged/chaincfg"
	"github.com/dogecoinw/doged/chaincfg/chainhash"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// consensusDecode parses consensus protocol transactions
func (e *Explorer) consensusDecode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.ConsensusInfo, error) {
	err := e.dbc.DB.Where("tx_hash = ?", tx.Txid).First(&models.ConsensusInfo{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("consensus already exist or err %s", tx.Txid)
	}

	param := &models.ConsensusInscription{}
	err = json.Unmarshal(pushedData, param)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	consensus, err := utils.ConvertConsensus(param)
	if err != nil {
		return nil, fmt.Errorf("ConvertConsensus err: %s", err.Error())
	}

	if len(tx.Vout) < 1 {
		return nil, fmt.Errorf("op error, vout length is not 0")
	}

	consensus.OrderId = uuid.New().String()
	consensus.FeeTxHash = tx.Vin[0].Txid
	consensus.TxHash = tx.Txid
	consensus.BlockHash = tx.BlockHash
	consensus.BlockNumber = number
	consensus.OrderStatus = 1

	consensus.HolderAddress = tx.Vout[0].ScriptPubKey.Addresses[0]

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

	if consensus.HolderAddress != txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0] {
		return nil, fmt.Errorf("the address is not the same as the previous transaction")
	}

	// Rules:
	// - stake: use current txhash as unique stake_id (for precise unstake reference)
	// - unstake: inscription must carry stake_id
	if consensus.Op == "stake" {
		consensus.StakeId = consensus.TxHash
	}
	if consensus.Op == "unstake" && consensus.StakeId == "" {
		return nil, fmt.Errorf("unstake requires stake_id")
	}

	err = e.dbc.DB.Save(consensus).Error
	if err != nil {
		return nil, fmt.Errorf("SaveConsensus err: %s", err.Error())
	}

	return consensus, nil
}

// consensusStake handles stake operations
func (e *Explorer) consensusStake(consensus *models.ConsensusInfo) error {
	// Build special address based on transaction hash
	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(consensus.TxHash+"--CONSENSUS"), &chaincfg.MainNetParams)

	tx := e.dbc.DB.Begin()
	err := e.dbc.ConsensusStake(tx, consensus, reservesAddress.String())
	if err != nil {
		tx.Rollback()
		return err
	}

	// Update status
	err = tx.Model(&models.ConsensusInfo{}).Where("tx_hash = ?", consensus.TxHash).Update("order_status", 0).Error
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

// consensusUnstake handles unstake operations
func (e *Explorer) consensusUnstake(consensus *models.ConsensusInfo) error {
	// Unstake precisely locates independent stake: find corresponding record via stake_id, get reserves_address
	if consensus.StakeId == "" {
		return fmt.Errorf("unstake requires stake_id")
	}
	record := &models.ConsensusStakeRecord{}
	if err := e.dbc.DB.Where("stake_id = ? AND status = ?", consensus.StakeId, "active").First(record).Error; err != nil {
		return fmt.Errorf("query consensus stake record error: %v", err)
	}

	tx := e.dbc.DB.Begin()
	err := e.dbc.ConsensusUnstake(tx, consensus, record.ReservesAddress)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Update status and amount (use amount from original stake record)
	err = tx.Model(&models.ConsensusInfo{}).Where("tx_hash = ?", consensus.TxHash).Updates(map[string]interface{}{
		"order_status": 0,
		"amt":          record.Amt,
	}).Error
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

func (e *Explorer) consensusFork(tx *gorm.DB, height int64) error {
	// Rollback consensus: read rollback logs, execute from back to front
	var reverts []*models.ConsensusRevert
	err := tx.Model(&models.ConsensusRevert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&reverts).Error
	if err != nil {
		return fmt.Errorf("FindConsensusRevert error: %v", err)
	}

	for _, r := range reverts {
		// stake: rollback should revoke stake (fund migration + delete record)
		if r.Op == "stake" {
			if err := e.dbc.ConsensusRevertStake(tx, r.FromAddress, r.ToAddress, r.Amt.Int(), r.TxHash, r.BlockNumber); err != nil {
				return fmt.Errorf("consensus fork revert stake error: %v", err)
			}
		}
		// unstake: rollback should revoke unstake (fund migration + reopen record)
		if r.Op == "unstake" {
			if err := e.dbc.ConsensusRevertUnstake(tx, r.ToAddress, r.FromAddress, r.Amt.Int(), r.TxHash, r.BlockNumber); err != nil {
				return fmt.Errorf("consensus fork revert unstake error: %v", err)
			}
		}
	}

	return nil
}
