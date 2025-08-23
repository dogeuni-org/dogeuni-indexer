package explorer

import (
	"dogeuni-indexer/models"
	"errors"
	"fmt"
	"github.com/dogecoinw/go-dogecoin/log"
	"gorm.io/gorm"
)

func (e *Explorer) forkBack() error {

	height := e.currentHeight

	blockHash, err := e.node.GetBlockHash(e.currentHeight)
	if err != nil {
		return err
	}

	block, err := e.node.GetBlockVerboseBool(blockHash)
	if err != nil {
		return err
	}

	localHash := ""
	err = e.dbc.DB.Model(&models.Block{}).Where("block_number = ?", height-1).Select("block_hash").First(&localHash).Error
	if err != nil {
		block0 := &models.Block{
			BlockNumber: height - 1,
			BlockHash:   block.PreviousHash,
		}
		err = e.dbc.DB.Create(block0).Error
		if err != nil {
			return err
		}
		return errors.New("localHash is nil")
	}

	if localHash != block.PreviousHash {
		log.Warn("forkBack Begin", "height", height)
		for blockHash.String() != localHash {
			height--
			blockHash, err = e.node.GetBlockHash(height)
			if err != nil {
				return fmt.Errorf("GetBlockHash error: %v", err)
			}

			err = e.dbc.DB.Model(&models.Block{}).Where("block_number = ?", height).Select("block_hash").First(&localHash).Error
			if localHash == "" {
				return errors.New("localHash is nil")
			}
		}

		tx := e.dbc.DB.Begin()
		err := e.fork(tx, height)
		if err != nil {
			log.Error("fork error", "err", err)
			tx.Rollback()
			return err
		}

		err = tx.Commit().Error
		if err != nil {
			return err
		}

		e.currentHeight = height
		log.Warn("forkBack End", "height", height)
	}

	return nil
}

func (e *Explorer) fork(tx *gorm.DB, height int64) error {

	err := e.delInfo(tx, height)
	if err != nil {
		return err
	}

	err = e.drc20Fork(tx, height)
	if err != nil {
		return err
	}

	err = e.meme20Fork(tx, height)
	if err != nil {
		return err
	}

	err = e.pumpFork(tx, height)
	if err != nil {
		return err
	}

	err = e.swapFork(tx, height)
	if err != nil {
		return err
	}

	err = e.swapV2Fork(tx, height)
	if err != nil {
		return err
	}

	err = e.fileFork(tx, height)
	if err != nil {
		return err
	}

	err = e.exchangeFork(tx, height)
	if err != nil {
		return err
	}

	err = e.stakeFork(tx, height)
	if err != nil {
		return err
	}

	err = e.boxFork(tx, height)
	if err != nil {
		return err
	}

	err = e.fileExchangeFork(tx, height)
	if err != nil {
		return err
	}

	err = e.stakeV2Fork(tx, height)
	if err != nil {
		return err
	}

	err = e.crossFork(tx, height)
	if err != nil {
		return err
	}

	err = e.inviteFork(tx, height)
	if err != nil {
		return err
	}

    // consensus
    err = e.consensusFork(tx, height)
    if err != nil {
        return err
    }

	err = e.delRevert(tx, height)
	if err != nil {
		return err
	}

	return nil

}

func (e *Explorer) delInfo(tx *gorm.DB, height int64) error {

	log.Info("delInfo", "height", height)

	err := tx.Where("block_number > ?", height).Delete(&models.Drc20Info{}).Error
	if err != nil {
		return fmt.Errorf("DeleteDrc20Info error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.SwapInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteDrc20Collect error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.WDogeInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteDrc20Revert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.NftInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteNftInfo error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.BoxInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteBoxInfo error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.ExchangeInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteExchangeInfo error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.StakeInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteStakeInfo error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.StakeRewardInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteStakeRewardInfo error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.BoxCollectAddress{}).Error
	if err != nil {
		return fmt.Errorf("DeleteBoxAddress error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.FileInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteFileInfo error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.FileExchangeInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteFileExchangeInfo error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.CrossInfo{}).Error
	if err != nil {
		return fmt.Errorf("CrossInfo error: %v", err)
	}

	// meme20
	err = tx.Where("block_number > ?", height).Delete(&models.Meme20Info{}).Error
	if err != nil {
		return fmt.Errorf("DeleteMeme20Info error: %v", err)
	}

	// swap-v2
	err = tx.Where("block_number > ?", height).Delete(&models.SwapV2Info{}).Error
	if err != nil {
		return fmt.Errorf("DeleteSwapV2Info error: %v", err)
	}

	// pump
	err = tx.Where("block_number > ?", height).Delete(&models.PumpInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeletePumpInfo error: %v", err)
	}

	// invite
	err = tx.Where("block_number > ?", height).Delete(&models.InviteInfo{}).Error
	if err != nil {
		return fmt.Errorf("DeleteInviteInfo error: %v", err)
	}
	return nil
}

func (e *Explorer) delRevert(tx *gorm.DB, height int64) error {

	err := tx.Where("block_number > ?", height).Delete(&models.Drc20Revert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteDrc20Revert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.SwapRevert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteSwapRevert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.FileRevert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteFileRevert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.ExchangeRevert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteExchangeRevert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.StakeRevert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteStakeRevert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.StakeRewardRevert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteStakeRewardRevert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.FileExchangeRevert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteFileExchangeRevert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.CrossRevert{}).Error
	if err != nil {
		return fmt.Errorf("CrossRevert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.Meme20Revert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteMeme20Revert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.SwapV2Revert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteSwapV2Revert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.PumpRevert{}).Error
	if err != nil {
		return fmt.Errorf("DeletePumpRevert error: %v", err)
	}

	err = tx.Where("block_number > ?", height).Delete(&models.InviteRevert{}).Error
	if err != nil {
		return fmt.Errorf("DeleteInviteRevert error: %v", err)
	}

    err = tx.Where("block_number > ?", height).Delete(&models.ConsensusRevert{}).Error
    if err != nil {
        return fmt.Errorf("DeleteConsensusRevert error: %v", err)
    }

	err = tx.Where("block_number > ?", height).Delete(&models.PumpInviteRewardRevert{}).Error
	if err != nil {
		return fmt.Errorf("DeletePumpInviteRewardRevert error: %v", err)
	}

	return nil
}
