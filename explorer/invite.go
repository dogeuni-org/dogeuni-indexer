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

// invite
func (e *Explorer) inviteDecode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.InviteInfo, error) {

	err := e.dbc.DB.Where("tx_hash = ?", tx.Hash).First(&models.InviteInfo{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("InviteInfo already exist or err %s", tx.Hash)
	}

	param := &models.InviteInscription{}
	err = json.Unmarshal(pushedData, param)
	if err != nil {
		return nil, fmt.Errorf("json Unmarshal err: %s", err.Error())
	}

	invite, err := utils.ConvertInvite(param)
	if err != nil {
		return nil, fmt.Errorf("ConvertInvite err: %s", err.Error())
	}

	invite.OrderId = uuid.New().String()
	invite.FeeTxHash = tx.Vin[0].Txid

	invite.TxHash = tx.Hash
	invite.BlockHash = tx.BlockHash
	invite.BlockNumber = number
	invite.OrderStatus = 1

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

	invite.HolderAddress = tx.Vout[0].ScriptPubKey.Addresses[0]
	if tx.Vout[0].Value != 0.001 {
		return nil, fmt.Errorf("the amount of tokens exceeds the 0.0001")
	}

	if invite.HolderAddress != txRawResult1.Vout[txRawResult0.Vin[0].Vout].ScriptPubKey.Addresses[0] {
		return nil, fmt.Errorf("the address is not the same as the previous transaction")
	}

	invite.FeeAddress = txRawResult0.Vout[tx.Vin[0].Vout].ScriptPubKey.Addresses[0]

	err = e.dbc.DB.Save(invite).Error
	if err != nil {
		return nil, fmt.Errorf("save err: %s", err.Error())
	}

	return invite, nil
}

func (e *Explorer) inviteDeploy(invite *models.InviteInfo) error {

	tx := e.dbc.DB.Begin()

	invitec := &models.InviteCollect{
		InviteAddress: invite.InviteAddress,
		HolderAddress: invite.HolderAddress,
	}

	err := tx.Create(invitec).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("save err: %s", err.Error())
	}

	inviteRevert := &models.InviteRevert{
		HolderAddress: invite.HolderAddress,
		InviteAddress: invite.InviteAddress,
		BlockNumber:   invite.BlockNumber,
	}

	err = tx.Create(inviteRevert).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("save err: %s", err.Error())
	}

	err = tx.Model(&models.InviteInfo{}).Where("tx_hash = ?", invite.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("update err: %s", err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("commit err: %s", err.Error())
	}

	return nil
}

func (e *Explorer) inviteFork(tx *gorm.DB, height int64) error {

	log.Info("fork", "invite", height)
	var inviteReverts []*models.InviteRevert
	err := tx.Model(&models.InviteRevert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&inviteReverts).Error

	if err != nil {
		return fmt.Errorf("meme20 revert error: %v", err)
	}

	for _, revert := range inviteReverts {
		err = tx.Where("invite_address = ? and holder_address = ?", revert.InviteAddress, revert.HolderAddress).Delete(&models.InviteCollect{}).Error
		if err != nil {
			return fmt.Errorf("delete err: %s", err.Error())
		}
	}

	return nil
}
