package storage

import (
	"dogeuni-indexer/models"
	"errors"
	"fmt"
	"github.com/dogecoinw/doged/btcutil"
	"github.com/dogecoinw/doged/chaincfg"
	"github.com/dogecoinw/go-dogecoin/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
)

const (
	FinishFeeAddress = "DJ9wVHBFnbcZUtfWdHWPEnijdxz1CABPUY"
	TxFeeAddress     = "D7NfMMzqWB9FaUssLwgCs14Q5F6CCfpf9A"
)

var (
	MemeMax             = models.NewNumber(100000000000000000)
	DogeInit            = models.NewNumber(300000000000)
	DogeMax             = models.NewNumber(10000000000000)
	DogeKingMax         = models.NewNumber(1000000000000)
	PumpFinishFee       = models.NewNumber(100000000000)
	PumpCreateHolderFee = models.NewNumber(10000000000)
)

func (db *DBClient) PumpDeploy(tx *gorm.DB, pump *models.PumpInfo) error {

	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(pump.Tick0Id+pump.Tick1Id), &chaincfg.MainNetParams)

	meme20c := &models.Meme20Collect{}
	err := tx.Where("tick_id = ?", pump.Tick0Id).First(meme20c).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("PumpDeploy First err: %s", err.Error())
	}

	if len(meme20c.TickId) == 0 {
		meme20c := &models.Meme20Collect{
			Tick:          pump.Symbol,
			TickId:        pump.Tick0Id,
			Name:          pump.Name,
			Logo:          pump.Logo,
			Max:           MemeMax,
			Dec:           8,
			Reserve:       pump.Reserve,
			HolderAddress: reservesAddress.String(),
		}

		err := tx.Create(meme20c).Error
		if err != nil {
			return fmt.Errorf("PumpDeploy Create err: %s", err.Error())
		}
	} else {
		updates := map[string]interface{}{
			"max_":    MemeMax,
			"tick":    pump.Symbol,
			"name":    pump.Name,
			"logo":    pump.Logo,
			"reserve": pump.Reserve,
			"dec_":    8,
		}

		err = tx.Model(meme20c).Updates(updates).Error
		if err != nil {
			return fmt.Errorf("PumpDeploy Updates err: %s", err.Error())
		}
	}

	if pump.Reserve > 0 {

		mememax := new(big.Int).Div(MemeMax.Int(), big.NewInt(100))
		mememax = new(big.Int).Mul(mememax, big.NewInt(int64(100-pump.Reserve)))

		meme20_0 := &models.Meme20CollectAddress{
			TickId:        pump.Tick0Id,
			HolderAddress: reservesAddress.String(),
			Amt:           (*models.Number)(mememax),
			Transactions:  1,
		}

		err = tx.Create(meme20_0).Error
		if err != nil {
			return fmt.Errorf("PumpDeploy Create err: %s", err.Error())
		}

		meme20_1 := &models.Meme20CollectAddress{
			TickId:        pump.Tick0Id,
			HolderAddress: pump.HolderAddress,
			Amt:           (*models.Number)(big.NewInt(0).Sub(MemeMax.Int(), mememax)),
			Transactions:  1,
		}

		err = tx.Create(meme20_1).Error
		if err != nil {
			return fmt.Errorf("PumpDeploy Create err: %s", err.Error())
		}

		sl := &models.PumpLiquidity{
			Tick0:           pump.Symbol,
			Tick0Id:         pump.Tick0Id,
			Amt0:            (*models.Number)(mememax),
			Tick1:           pump.Tick1Id,
			Tick1Id:         pump.Tick1Id,
			Amt1:            DogeInit,
			HolderAddress:   pump.HolderAddress,
			ReservesAddress: reservesAddress.String(),
			KingDate:        1,
		}

		err = tx.Create(sl).Error
		if err != nil {
			return fmt.Errorf("PumpLiquidity Create err: %s", err.Error())
		}

		pump.Amt0Out = (*models.Number)(mememax)
		pump.Amt1Out = DogeInit

	} else {

		meme20_0 := &models.Meme20CollectAddress{
			TickId:        pump.Tick0Id,
			HolderAddress: reservesAddress.String(),
			Amt:           MemeMax,
			Transactions:  1,
		}

		err = tx.Create(meme20_0).Error
		if err != nil {
			return fmt.Errorf("PumpDeploy Create err: %s", err.Error())
		}

		sl := &models.PumpLiquidity{
			Tick0:           pump.Symbol,
			Tick0Id:         pump.Tick0Id,
			Amt0:            MemeMax,
			Tick1:           pump.Tick1Id,
			Tick1Id:         pump.Tick1Id,
			Amt1:            DogeInit,
			HolderAddress:   pump.HolderAddress,
			ReservesAddress: reservesAddress.String(),
			KingDate:        1,
		}

		err = tx.Create(sl).Error
		if err != nil {
			return fmt.Errorf("PumpLiquidity Create err: %s", err.Error())
		}

		pump.Amt0Out = MemeMax
		pump.Amt1Out = DogeInit
	}

	err = db.SummaryPumpCreate(tx, pump)
	if err != nil {
		log.Error("SummaryPump error", "err", err)
	}

	revert := &models.PumpRevert{
		Op:          "deploy",
		TickId:      pump.Tick0Id,
		BlockNumber: pump.BlockNumber,
	}

	err = tx.Create(revert).Error
	if err != nil {
		return err
	}

	if pump.Amt1.Int().Cmp(big.NewInt(0)) > 0 {
		pumpt := &models.PumpInfo{
			Op:            "trade",
			PairId:        pump.TxHash,
			Tick0:         pump.Tick1,
			Tick0Id:       pump.Tick1Id,
			Tick1:         pump.Tick0,
			Tick1Id:       pump.Tick0Id,
			Amt0:          pump.Amt1,
			HolderAddress: pump.HolderAddress,
			TxHash:        pump.TxHash,
			BlockNumber:   pump.BlockNumber,
			BlockTime:     pump.BlockTime,
		}

		err = db.PumpTrade(tx, pumpt)
		if err != nil {
			return err
		}

		pump.Amt1Out = pumpt.Amt1Out
	}

	return nil
}

func (db *DBClient) PumpTrade(tx *gorm.DB, pump *models.PumpInfo) error {

	pumpl := &models.PumpLiquidity{}
	err := tx.Where("tick0_id = ? ", pump.PairId).First(pumpl).Error
	if err != nil {
		return fmt.Errorf("pumpTrade  error: %v", err)
	}

	amtMap := make(map[string]*big.Int)
	amtMap[pumpl.Tick0Id] = pumpl.Amt0.Int()
	amtMap[pumpl.Tick1Id] = pumpl.Amt1.Int()

	inviter := &models.InviteCollect{}
	err = tx.Where("holder_address = ?", pump.HolderAddress).First(inviter).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("pumpTrade error: %v", err)
	}

	amtfee0 := big.NewInt(0)
	if len(pump.Tick0Id) < MEMETICKID_LENGTH {
		amtfee0 = new(big.Int).Div(pump.Amt0.Int(), big.NewInt(100))
		err = db.TransferDrc20(tx, pump.Tick0Id, pump.HolderAddress, TxFeeAddress, amtfee0, pump.TxHash, pump.BlockNumber, false)
		if err != nil {
			return err
		}

		if len(inviter.InviteAddress) > 0 {
			amtfee1 := new(big.Int).Div(amtfee0, big.NewInt(5))
			err = db.TransferDrc20(tx, pump.Tick0Id, TxFeeAddress, inviter.InviteAddress, amtfee1, pump.TxHash, pump.BlockNumber, false)
			if err != nil {
				return err
			}

			err = db.InvitePump(tx, inviter, amtfee1, pump.BlockNumber)
			if err != nil {
				return err
			}
		}
	}

	amtin := new(big.Int).Sub(pump.Amt0.Int(), amtfee0)
	amtout := new(big.Int).Mul(amtin, amtMap[pump.Tick1Id])
	amtout = new(big.Int).Div(amtout, new(big.Int).Add(amtMap[pump.Tick0Id], amtin))

	if len(pump.Tick0Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, pump.Tick0Id, pump.HolderAddress, pumpl.ReservesAddress, amtin, pump.TxHash, pump.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, pump.Tick0Id, pump.HolderAddress, pumpl.ReservesAddress, amtin, pump.TxHash, pump.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	amtout1 := *amtout

	if len(pump.Tick1Id) < MEMETICKID_LENGTH {
		amtfee1 := new(big.Int).Div(amtout, big.NewInt(100))
		err = db.TransferDrc20(tx, pump.Tick1Id, pumpl.ReservesAddress, TxFeeAddress, amtfee1, pump.TxHash, pump.BlockNumber, false)
		if err != nil {
			return err
		}

		amtout = new(big.Int).Sub(amtout, amtfee1)

		if len(inviter.InviteAddress) > 0 {
			amtfee2 := new(big.Int).Div(amtfee1, big.NewInt(5))
			err = db.TransferDrc20(tx, pump.Tick1Id, TxFeeAddress, inviter.InviteAddress, amtfee2, pump.TxHash, pump.BlockNumber, false)
			if err != nil {
				return err
			}

			err = db.InvitePump(tx, inviter, amtfee2, pump.BlockNumber)
			if err != nil {
				return err
			}
		}
	}

	if len(pump.Tick1Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, pump.Tick1Id, pumpl.ReservesAddress, pump.HolderAddress, amtout, pump.TxHash, pump.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, pump.Tick1Id, pumpl.ReservesAddress, pump.HolderAddress, amtout, pump.TxHash, pump.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	pump.Amt1Out = (*models.Number)(amtout)

	revert := &models.PumpRevert{
		Op:          "trade",
		TickId:      pumpl.Tick0Id,
		Amt0:        pumpl.Amt0,
		Amt1:        pumpl.Amt1,
		BlockNumber: pump.BlockNumber,
	}

	err = tx.Create(revert).Error
	if err != nil {
		return err
	}

	if pump.Tick0Id == pumpl.Tick0Id {
		pump.Tick0 = pumpl.Tick0
		pump.Tick1 = pumpl.Tick1

		pumpl.Amt0 = (*models.Number)(new(big.Int).Add(pumpl.Amt0.Int(), amtin))
		pumpl.Amt1 = (*models.Number)(new(big.Int).Sub(pumpl.Amt1.Int(), &amtout1))
	} else {
		pump.Tick0 = pumpl.Tick1
		pump.Tick1 = pumpl.Tick0

		pumpl.Amt0 = (*models.Number)(new(big.Int).Sub(pumpl.Amt0.Int(), &amtout1))
		pumpl.Amt1 = (*models.Number)(new(big.Int).Add(pumpl.Amt1.Int(), amtin))
	}

	meme := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", pumpl.Tick1Id, pumpl.ReservesAddress).First(meme).Error
	if err != nil {
		return fmt.Errorf("pumpTrade error: %v", err)
	}

	kingDate := pumpl.KingDate
	if meme.AmtSum.Int().Cmp(DogeKingMax.Int()) >= 0 && pumpl.KingDate == 1 {
		kingDate = models.LocalTime(pump.BlockTime)
	}

	updates := map[string]interface{}{
		"amt0":      pumpl.Amt0,
		"amt1":      pumpl.Amt1,
		"king_date": kingDate,
	}

	err = tx.Model(pumpl).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("pumpTrade  error: %v", err)
	}

	if meme.AmtSum.Int().Cmp(DogeMax.Int()) >= 0 {
		err = db.PumpFinish(tx, pump, pumpl)
		if err != nil {
			return err
		}
	}

	//go func() {
	pump.Amt0 = (*models.Number)(amtin)
	pump.Amt1 = (*models.Number)(amtout)
	err = db.SummaryPump(tx, pump)
	if err != nil {
		log.Error("SummaryPump error", "err", err)
	}
	//}()

	return nil
}

func (db *DBClient) PumpFinish(tx *gorm.DB, pump *models.PumpInfo, pumpl *models.PumpLiquidity) error {

	if len(pumpl.Tick1Id) < MEMETICKID_LENGTH {

		err := db.TransferDrc20(tx, pumpl.Tick1Id, pumpl.ReservesAddress, FinishFeeAddress, PumpFinishFee.Int(), pump.TxHash, pump.BlockNumber, false)
		if err != nil {
			return err
		}

		wdoge := &models.WDogeInfo{}
		wdoge.OrderId = uuid.New().String()
		wdoge.Op = "withdraw-pump"
		wdoge.Tick = "WDOGE(WRAPPED-DOGE)"
		wdoge.Amt = PumpFinishFee
		wdoge.HolderAddress = FinishFeeAddress
		wdoge.TxHash = pump.TxHash
		wdoge.BlockHash = pump.BlockHash
		wdoge.BlockNumber = pump.BlockNumber
		err = db.wdogeWithdrawPump(tx, wdoge)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("wdogeWithdrawPump err: %s", err.Error())
		}

		err = db.TransferDrc20(tx, pumpl.Tick1Id, pumpl.ReservesAddress, pumpl.HolderAddress, PumpCreateHolderFee.Int(), pump.TxHash, pump.BlockNumber, false)
		if err != nil {
			return err
		}

		wdoge = &models.WDogeInfo{}
		wdoge.OrderId = uuid.New().String()
		wdoge.Op = "withdraw-pump"
		wdoge.Tick = "WDOGE(WRAPPED-DOGE)"
		wdoge.Amt = PumpCreateHolderFee
		wdoge.HolderAddress = pumpl.HolderAddress
		wdoge.TxHash = pump.TxHash
		wdoge.BlockHash = pump.BlockHash
		wdoge.BlockNumber = pump.BlockNumber
		err = db.wdogeWithdrawPump(tx, wdoge)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("wdogeWithdrawPump err: %s", err.Error())
		}
	}

	var mca0 *models.Meme20CollectAddress
	var mca1 *models.Drc20CollectAddress
	err := tx.Where("tick_id = ? and holder_address = ?", pumpl.Tick0Id, pumpl.ReservesAddress).First(&mca0).Error
	if err != nil {
		return fmt.Errorf("pumpFinish error: %v", err)
	}

	err = tx.Where("tick = ? and holder_address = ?", pumpl.Tick1Id, pumpl.ReservesAddress).First(&mca1).Error
	if err != nil {
		return fmt.Errorf("pumpFinish error: %v", err)
	}

	swap := &models.SwapV2Info{
		Op:            "create-pump-finish",
		PairId:        pump.TxHash,
		Tick0Id:       pumpl.Tick0Id,
		Tick0:         pumpl.Tick0,
		Tick1Id:       pumpl.Tick1Id,
		Tick1:         pumpl.Tick1,
		Amt0:          mca0.Amt,
		Amt1:          mca1.AmtSum,
		HolderAddress: pumpl.ReservesAddress,
		TxHash:        pump.TxHash,
		BlockNumber:   pump.BlockNumber,
	}

	swap.Amt0Out = mca0.Amt
	swap.Amt1Out = mca1.AmtSum

	err = db.SwapV2Create(tx, swap)
	if err != nil {
		return fmt.Errorf("swapCreate SwapCreate error: %v", err)
	}

	return nil
}

func (db *DBClient) wdogeWithdrawPump(tx *gorm.DB, wdoge *models.WDogeInfo) error {

	err := db.DogeWithdraw(tx, wdoge)
	if err != nil {
		return err
	}

	err = tx.Create(wdoge).Error
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) InvitePump(tx *gorm.DB, invite *models.InviteCollect, amt *big.Int, height int64) error {

	inviteReward := &models.PumpInviteReward{}
	err := tx.Where("holder_address = ?", invite.HolderAddress).First(inviteReward).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("InvitePump error: %v", err)
		}

		inviteReward.InviteAddress = invite.InviteAddress
		inviteReward.HolderAddress = invite.HolderAddress
		inviteReward.InviteReward = (*models.Number)(big.NewInt(0))
		err := tx.Create(inviteReward).Error
		if err != nil {
			return fmt.Errorf("InvitePump error: %v", err)
		}
	}

	revert := &models.PumpInviteRewardRevert{
		InviteAddress: invite.InviteAddress,
		HolderAddress: invite.HolderAddress,
		InviteReward:  inviteReward.InviteReward,
		BlockNumber:   height,
	}

	err = tx.Create(revert).Error
	if err != nil {
		return fmt.Errorf("invitePumpRevert error: %v", err)
	}

	count := inviteReward.InviteReward.Int()
	sum := big.NewInt(0).Add(count, amt)

	err = tx.Model(inviteReward).Update("invite_reward", sum.String()).Error
	if err != nil {
		return fmt.Errorf("invitePump error: %v", err)
	}

	return nil

}

func (db *DBClient) FindPumpPriceAll() ([]*SwapV2Price, int64, error) {

	liquidityAll := make([]*models.PumpLiquidity, 0)

	err := db.DB.Find(&liquidityAll).Error
	if err != nil {
		return nil, 0, err
	}

	liquidityMap := make(map[string]*big.Float)
	for _, v := range liquidityAll {

		if v.Amt1.Int().Cmp(big.NewInt(10300000000000)) >= 0 {
			continue
		}

		liquidityMap[v.Tick0Id] = new(big.Float).Quo(new(big.Float).SetInt(v.Amt1.Int()), new(big.Float).SetInt(v.Amt0.Int()))
	}

	liquiditys := make([]*SwapV2Price, 0)
	lenght := int64(0)
	for k, v := range liquidityMap {

		f := new(big.Float).Mul(v, big.NewFloat(1e18))
		fi, _ := big.NewInt(0).SetString(f.Text('f', 0), 10)
		liquiditys = append(liquiditys, &SwapV2Price{
			TickId:    k,
			LastPrice: fi.String(),
		})
		lenght++
	}

	return liquiditys, lenght, nil
}
