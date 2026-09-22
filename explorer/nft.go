package explorer

import (
	"bytes"
	"dogeuni-indexer/models"
	"dogeuni-indexer/utils"
	"errors"
	"fmt"
	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/chaincfg/chainhash"
	"github.com/dogecoinw/go-dogecoin/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (e *Explorer) nftDecode(tx *btcjson.TxRawResult, number int64) (*models.NftInfo, error) {

	if err := e.dropPending(&models.NftInfo{}, tx.Hash); err != nil {
		return nil, err
	}

	err := e.dbc.DB.Where("tx_hash = ?", tx.Hash).First(&models.NftInfo{}).Error
	if err == nil {
		return nil, fmt.Errorf("nft already exist %s", tx.Hash)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: dedup: %v", STORAGE_ERR, err)
	}

	param, err := e.reDecodeNft(tx)
	if err != nil {
		return nil, fmt.Errorf("reDecodeNft err: %s", err.Error())
	}

	nft, err := utils.ConvertNft(param)
	if err != nil {
		return nil, fmt.Errorf("ConvertNft err: %s", err.Error())
	}

	nft.OrderId = uuid.New().String()
	nft.FeeTxHash = tx.Vin[0].Txid

	nft.TxHash = tx.Hash
	nft.BlockHash = tx.BlockHash
	nft.BlockNumber = number
	nft.OrderStatus = 1

	if nft.Op == "deploy" {

		if len(tx.Vout) != 2 {
			return nil, errors.New("deploy op error, vout length is not 2")
		}

		nft.HolderAddress, err = outputAddress(tx, 0)
		if err != nil {
			return nil, err
		}

		if tx.Vout[0].Value != 0.001 {
			return nil, fmt.Errorf("The amount of tokens exceeds the 0.0001")
		}

		if tx.Vout[1].Value < 1000 {
			return nil, fmt.Errorf("The balance is insufficient")
		}

		if addr, err := outputAddress(tx, 1); err != nil {
			return nil, err
		} else if addr != nftFeeAddress {
			return nil, fmt.Errorf("The address is incorrect")
		}
	}

	if nft.Op == "mint" {

		if len(tx.Vout) != 2 {
			return nil, errors.New("mint op error, vout length is not 2")
		}

		nft.HolderAddress, err = outputAddress(tx, 0)
		if err != nil {
			return nil, err
		}

		if tx.Vout[0].Value != 0.001 {
			return nil, fmt.Errorf("The amount of tokens exceeds the 0.0001")
		}

		if tx.Vout[1].Value < 10 {
			return nil, fmt.Errorf("The balance is insufficient")
		}

		if addr, err := outputAddress(tx, 1); err != nil {
			return nil, err
		} else if addr != nftFeeAddress {
			return nil, fmt.Errorf("The address is incorrect")
		}
	}

	txHash0, _ := chainhash.NewHashFromStr(tx.Vin[0].Txid)
	txRawResult0, err := e.node.GetRawTransactionVerboseBool(txHash0)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", CHAIN_NETWORK_ERR, err)
	}

	if nft.Op == "transfer" {

		txhash1, _ := chainhash.NewHashFromStr(txRawResult0.Vin[0].Txid)
		txRawResult1, err := e.node.GetRawTransactionVerboseBool(txhash1)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", CHAIN_NETWORK_ERR, err)
		}

		nft.HolderAddress, err = outputAddress(txRawResult1, int(txRawResult0.Vin[0].Vout))
		if err != nil {
			return nil, err
		}
		nft.ToAddress, err = outputAddress(tx, 0)
		if err != nil {
			return nil, err
		}

		if nft.HolderAddress == nft.ToAddress {
			return nil, errors.New("The address is the same")
		}
	}

	nft.FeeAddress, err = outputAddress(txRawResult0, int(tx.Vin[0].Vout))
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(nft.ImageData)
	hash, _ := e.ipfs.Add(reader)
	nft.ImagePath = "https://ipfs.unielon.com/ipfs/" + hash

	err = e.dbc.DB.Create(nft).Error
	if err != nil {
		return nil, fmt.Errorf("%w: save inscription: %v", STORAGE_ERR, err)
	}

	return nft, nil
}

func (e *Explorer) nftDeploy(nft *models.NftInfo) error {
	log.Info("explorer", "p", "nft/ai", "op", "deploy", "tx_hash", nft.TxHash)

	tx := e.dbc.DB.Begin()
	err := e.dbc.NftDeploy(tx, nft)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.NftInfo{}).Where("tx_hash = ?", nft.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("nftDeploy commit err: %s order_id: %s", err.Error(), nft.OrderId)
	}

	return nil
}

func (e *Explorer) nftMint(nft *models.NftInfo) error {

	log.Info("explorer", "p", "nft/ai", "op", "mint", "tx_hash", nft.TxHash)
	tx := e.dbc.DB.Begin()

	err := e.dbc.NftMint(tx, nft)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.NftInfo{}).Where("tx_hash = ?", nft.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("nftMint commit err: %s order_id: %s", err.Error(), nft.OrderId)
	}

	return nil
}

func (e *Explorer) nftTransfer(nft *models.NftInfo) error {

	log.Info("explorer", "p", "nft/ai", "op", "transfer", "tx_hash", nft.TxHash)

	tx := e.dbc.DB.Begin()
	err := e.dbc.NftTransfer(tx, nft)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Model(&models.NftInfo{}).Where("tx_hash = ?", nft.TxHash).Update("order_status", 0).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("nftTransfer commit err: %s order_id: %s", err.Error(), nft.OrderId)
	}

	return nil
}

func (e *Explorer) nftFork(tx *gorm.DB, height int64) error {
	log.Info("fork", "nft", height)

	var nftReverts []*models.NftRevert
	err := tx.Model(&models.NftRevert{}).
		Where("block_number > ?", height).
		Order("id desc").
		Find(&nftReverts).Error
	if err != nil {
		return fmt.Errorf("FindNftRevert error: %v", err)
	}

	for _, revert := range nftReverts {
		switch revert.Op {
		case "deploy":
			err = tx.Where("tick = ?", revert.Tick).Delete(&models.NftCollect{}).Error
			if err != nil {
				return fmt.Errorf("nftFork deploy error: %v", err)
			}

		case "mint":
			err = tx.Where("tick = ? AND tick_id = ?", revert.Tick, revert.TickId).Delete(&models.NftCollectAddress{}).Error
			if err != nil {
				return fmt.Errorf("nftFork mint error: %v", err)
			}

			err = tx.Model(&models.NftCollect{}).Where("tick = ?", revert.Tick).Updates(map[string]interface{}{
				"transactions": gorm.Expr("transactions - 1"),
				"tick_sum":     gorm.Expr("tick_sum - 1"),
			}).Error
			if err != nil {
				return fmt.Errorf("nftFork mint collect error: %v", err)
			}

		case "transfer":
			err = e.dbc.TransferNft(tx, revert.Tick, revert.ToAddress, revert.FromAddress, revert.TickId, height, true)
			if err != nil {
				return fmt.Errorf("nftFork transfer error: %v", err)
			}

		default:
			return fmt.Errorf("nftFork unknown op %q id %d", revert.Op, revert.ID)
		}
	}

	return nil
}
