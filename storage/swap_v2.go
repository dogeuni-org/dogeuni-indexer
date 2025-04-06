package storage

import (
	"dogeuni-indexer/models"
	"fmt"
	"github.com/dogecoinw/doged/btcutil"
	"github.com/dogecoinw/doged/chaincfg"
	"github.com/dogecoinw/go-dogecoin/log"
	"gorm.io/gorm"
	"math/big"
)

const (
	MEMETICKID_LENGTH = 64
)

func (db *DBClient) SwapV2Create(tx *gorm.DB, swap *models.SwapV2Info) error {

	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(swap.PairId), &chaincfg.MainNetParams)

	liquidityBase := new(big.Int).Sqrt(new(big.Int).Mul(swap.Amt0.Int(), swap.Amt1.Int()))
	if liquidityBase.Cmp(big.NewInt(MINI_LIQUIDITY)) > 0 {
		liquidityBase = new(big.Int).Sub(liquidityBase, big.NewInt(MINI_LIQUIDITY))
	} else {
		return fmt.Errorf("add liquidity must be greater than MINI_LIQUIDITY firstly")
	}

	swap.Amt0Out = swap.Amt0
	swap.Amt1Out = swap.Amt1
	swap.Liquidity = (*models.Number)(liquidityBase)

	sl := &models.SwapV2Liquidity{
		PairId:          swap.PairId,
		Tick0:           swap.Tick0,
		Tick0Id:         swap.Tick0Id,
		Tick1:           swap.Tick1,
		Tick1Id:         swap.Tick1Id,
		HolderAddress:   swap.HolderAddress,
		ReservesAddress: reservesAddress.String(),
		LiquidityTotal:  (*models.Number)(liquidityBase),
	}

	err := tx.Create(sl).Error
	if err != nil {
		return fmt.Errorf("SwapCreate Create err: %s", err.Error())
	}

	if len(swap.Tick0Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, swap.Tick0Id, swap.HolderAddress, reservesAddress.String(), swap.Amt0.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, swap.Tick0Id, swap.HolderAddress, reservesAddress.String(), swap.Amt0.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	if len(swap.Tick1Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, swap.Tick1Id, swap.HolderAddress, reservesAddress.String(), swap.Amt1.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, swap.Tick1Id, swap.HolderAddress, reservesAddress.String(), swap.Amt1.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	meme20c := &models.Meme20Collect{
		Tick:          "Liquidity Provider",
		TickId:        swap.PairId,
		Dec:           8,
		HolderAddress: reservesAddress.String(),
	}

	err = tx.Create(meme20c).Error
	if err != nil {
		return fmt.Errorf("SwapV2Create Create err: %s", err.Error())
	}

	err = db.MintMeme20(tx, swap.PairId, swap.HolderAddress, liquidityBase, swap.TxHash, swap.BlockNumber, false)
	if err != nil {
		return err
	}

	err = db.MintMeme20(tx, swap.PairId, reservesAddress.String(), big.NewInt(MINI_LIQUIDITY), swap.TxHash, swap.BlockNumber, false)
	if err != nil {
		return err
	}

	err = db.UpdateV2Liquidity(tx, swap.PairId)
	if err != nil {
		return err
	}

	revert := &models.SwapV2Revert{
		Op:          "create",
		PairId:      swap.PairId,
		BlockNumber: swap.BlockNumber,
	}

	err = tx.Create(revert).Error
	if err != nil {
		return fmt.Errorf("SwapV2Revert Create err: %s", err.Error())
	}

	return nil
}

func (db *DBClient) SwapV2Add(tx *gorm.DB, swap *models.SwapV2Info) error {

	reservesAddress, _ := btcutil.NewAddressScriptHash([]byte(swap.PairId), &chaincfg.MainNetParams)

	amt0Out := big.NewInt(0)
	amt1Out := big.NewInt(0)

	swapl := &models.SwapV2Liquidity{}
	err := tx.Where("pair_id = ?", swap.PairId).First(swapl).Error
	if err != nil {
		return fmt.Errorf("swapRemove FindSwapLiquidity error: %v", err)
	}

	swap.Tick0Id = swapl.Tick0Id
	swap.Tick0 = swapl.Tick0
	swap.Tick1Id = swapl.Tick1Id
	swap.Tick1 = swapl.Tick1

	amountBOptimal := big.NewInt(0).Mul(swap.Amt0.Int(), swapl.Amt1.Int())
	amountBOptimal = big.NewInt(0).Div(amountBOptimal, swapl.Amt0.Int())
	if amountBOptimal.Cmp(swap.Amt1Min.Int()) >= 0 && swap.Amt1.Int().Cmp(amountBOptimal) >= 0 {
		amt0Out = swap.Amt0.Int()
		amt1Out = amountBOptimal
	} else {
		amountAOptimal := big.NewInt(0).Mul(swap.Amt1.Int(), swapl.Amt0.Int())
		amountAOptimal = big.NewInt(0).Div(amountAOptimal, swapl.Amt1.Int())
		if amountAOptimal.Cmp(swap.Amt0Min.Int()) >= 0 && swap.Amt0.Int().Cmp(amountAOptimal) >= 0 {
			amt0Out = amountAOptimal
			amt1Out = swap.Amt1.Int()
		} else {
			return fmt.Errorf("the amount of tokens exceeds the balance")
		}
	}

	liquidity0 := new(big.Int).Mul(amt0Out, swapl.LiquidityTotal.Int())
	liquidity0 = new(big.Int).Div(liquidity0, swapl.Amt0.Int())

	liquidity1 := new(big.Int).Mul(amt1Out, swapl.LiquidityTotal.Int())
	liquidity1 = new(big.Int).Div(liquidity1, swapl.Amt1.Int())

	liquidity := liquidity0
	if liquidity0.Cmp(liquidity1) > 0 {
		liquidity = liquidity1
	}

	swap.Amt0Out = (*models.Number)(amt0Out)
	swap.Amt1Out = (*models.Number)(amt1Out)

	if len(swapl.Tick0Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, swapl.Tick0Id, swap.HolderAddress, reservesAddress.String(), swap.Amt0.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, swapl.Tick0Id, swap.HolderAddress, reservesAddress.String(), swap.Amt0.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	if len(swapl.Tick1Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, swapl.Tick1Id, swap.HolderAddress, reservesAddress.String(), swap.Amt1.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, swapl.Tick1Id, swap.HolderAddress, reservesAddress.String(), swap.Amt1.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	err = db.MintMeme20(tx, swapl.PairId, swap.HolderAddress, liquidity, swap.TxHash, swap.BlockNumber, false)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = db.UpdateV2Liquidity(tx, swap.PairId)
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (db *DBClient) SwapV2Remove(tx *gorm.DB, swap *models.SwapV2Info) error {

	swapl := &models.SwapV2Liquidity{}
	err := tx.Where("pair_id = ?", swap.PairId).First(swapl).Error
	if err != nil {
		return fmt.Errorf("swapRemove FindSwapLiquidity error: %v", err)
	}

	swap.Tick0 = swapl.Tick0
	swap.Tick0Id = swapl.Tick0Id
	swap.Tick1 = swapl.Tick1
	swap.Tick1Id = swapl.Tick1Id

	amt0Out := new(big.Int).Mul(swap.Liquidity.Int(), swapl.Amt0.Int())
	amt0Out = new(big.Int).Div(amt0Out, swapl.LiquidityTotal.Int())

	amt1Out := new(big.Int).Mul(swap.Liquidity.Int(), swapl.Amt1.Int())
	amt1Out = new(big.Int).Div(amt1Out, swapl.LiquidityTotal.Int())

	if swapl.Amt0.Int().Cmp(amt0Out) < 0 || swapl.Amt1.Int().Cmp(amt1Out) < 0 {
		return fmt.Errorf("swapRemove FindSwapLiquidity error: %v", err)
	}

	swap.Amt0Out = (*models.Number)(amt0Out)
	swap.Amt1Out = (*models.Number)(amt1Out)

	if len(swapl.Tick0Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, swapl.Tick0Id, swapl.ReservesAddress, swap.HolderAddress, amt0Out, swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, swapl.Tick0Id, swapl.ReservesAddress, swap.HolderAddress, amt0Out, swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	if len(swapl.Tick1Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, swapl.Tick1Id, swapl.ReservesAddress, swap.HolderAddress, amt1Out, swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, swapl.Tick1Id, swapl.ReservesAddress, swap.HolderAddress, amt1Out, swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	err = db.BurnMeme20(tx, swapl.PairId, swap.HolderAddress, swap.Liquidity.Int(), swap.TxHash, swap.BlockNumber, false)
	if err != nil {
		return err
	}

	err = db.UpdateV2Liquidity(tx, swap.PairId)
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) SwapV2Exec(tx *gorm.DB, swap *models.SwapV2Info) error {

	swapl := &models.SwapV2Liquidity{}
	err := tx.Where("pair_id = ?", swap.PairId).First(swapl).Error
	if err != nil {
		return fmt.Errorf("swapRemove FindSwapLiquidity error: %v", err)
	}

	if swap.Tick0Id == swapl.Tick0Id {
		swap.Tick0 = swapl.Tick0
		swap.Tick1 = swapl.Tick1
		swap.Tick1Id = swapl.Tick1Id
	} else if swap.Tick0Id == swapl.Tick1Id {
		swap.Tick1Id = swapl.Tick0Id
		swap.Tick0 = swapl.Tick1
		swap.Tick1 = swapl.Tick0
	} else {
		return fmt.Errorf("the contract does not exist err")
	}

	amtMap := make(map[string]*big.Int)
	amtMap[swapl.Tick0Id] = swapl.Amt0.Int()
	amtMap[swapl.Tick1Id] = swapl.Amt1.Int()

	amtfee0 := new(big.Int).Div(swap.Amt0.Int(), big.NewInt(1000))
	amtin := new(big.Int).Mul(amtfee0, big.NewInt(3))
	amtin = new(big.Int).Sub(swap.Amt0.Int(), amtin)

	amtout := new(big.Int).Mul(amtin, amtMap[swap.Tick1Id])
	amtout = new(big.Int).Div(amtout, new(big.Int).Add(amtMap[swap.Tick0Id], amtin))

	swap.Amt1Out = (*models.Number)(amtout)

	if len(swap.Tick0Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, swap.Tick0Id, swap.HolderAddress, swapl.ReservesAddress, swap.Amt0.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, swap.Tick0Id, swap.HolderAddress, swapl.ReservesAddress, swap.Amt0.Int(), swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	if len(swap.Tick1Id) < MEMETICKID_LENGTH {
		err = db.TransferDrc20(tx, swap.Tick1Id, swapl.ReservesAddress, swap.HolderAddress, amtout, swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	} else {
		err = db.TransferMeme20(tx, swap.Tick1Id, swapl.ReservesAddress, swap.HolderAddress, amtout, swap.TxHash, swap.BlockNumber, false)
		if err != nil {
			return err
		}
	}

	err = db.UpdateV2Liquidity(tx, swap.PairId)
	if err != nil {
		return err
	}

	go func() {
		err = db.SummarySwapV2(swap)
		if err != nil {
			log.Error("SummaryPump error", "err", err)
		}
	}()

	return nil
}

func (db *DBClient) UpdateV2Liquidity(tx *gorm.DB, tickId string) error {

	// tick0
	err := tx.Exec(`UPDATE swap_v2_liquidity
				SET amt0 = (
					SELECT b.amt_sum
					FROM drc20_collect_address b
					WHERE 
						swap_v2_liquidity.tick0_id = b.tick AND 
						swap_v2_liquidity.reserves_address = b.holder_address
				)
				WHERE EXISTS (
					SELECT 1
					FROM drc20_collect_address b
					WHERE 
						swap_v2_liquidity.tick0_id = b.tick AND 
						swap_v2_liquidity.reserves_address = b.holder_address AND 
						swap_v2_liquidity.pair_id = ?
				)`, tickId).Error
	if err != nil {
		return fmt.Errorf("UpdateLiquidity error: %s", err.Error())
	}

	err = tx.Exec(`UPDATE swap_v2_liquidity
				SET amt0 = (
					SELECT b.amt
					FROM meme20_collect_address b
					WHERE 
						swap_v2_liquidity.tick0_id = b.tick_id AND 
						swap_v2_liquidity.reserves_address = b.holder_address
				)
				WHERE EXISTS (
					SELECT 1
					FROM meme20_collect_address b
					WHERE 
						swap_v2_liquidity.tick0_id = b.tick_id AND 
						swap_v2_liquidity.reserves_address = b.holder_address AND 
						swap_v2_liquidity.pair_id = ?
				)`, tickId).Error
	if err != nil {
		return fmt.Errorf("UpdateLiquidity error: %s", err.Error())
	}

	// tick1
	err = tx.Exec(` UPDATE swap_v2_liquidity
				SET amt1 = (
					SELECT b.amt_sum
					FROM drc20_collect_address b
					WHERE 
						swap_v2_liquidity.tick1_id = b.tick AND 
						swap_v2_liquidity.reserves_address = b.holder_address
				)
				WHERE EXISTS (
					SELECT 1
					FROM drc20_collect_address b
					WHERE 
						swap_v2_liquidity.tick1_id = b.tick AND 
						swap_v2_liquidity.reserves_address = b.holder_address AND 
						swap_v2_liquidity.pair_id = ?
				)`, tickId).Error
	if err != nil {
		return err
	}

	err = tx.Exec(` UPDATE swap_v2_liquidity
				SET amt1 = (
					SELECT b.amt
					FROM meme20_collect_address b
					WHERE 
						swap_v2_liquidity.tick1_id = b.tick_id AND 
						swap_v2_liquidity.reserves_address = b.holder_address
				)
				WHERE EXISTS (
					SELECT 1
					FROM meme20_collect_address b
					WHERE 
						swap_v2_liquidity.tick1_id = b.tick_id AND 
						swap_v2_liquidity.reserves_address = b.holder_address AND 
						swap_v2_liquidity.pair_id = ?
				)`, tickId).Error
	if err != nil {
		return err
	}

	// pair
	err = tx.Exec(`UPDATE swap_v2_liquidity
				SET liquidity_total = (
					SELECT b.max_
					FROM meme20_collect b
					WHERE swap_v2_liquidity.pair_id = b.tick_id
				)
				WHERE pair_id = ?`, tickId).Error
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) UpdateV2LiquidityFork(tx *gorm.DB) error {

	// tick0
	err := tx.Exec(`UPDATE swap_v2_liquidity
				SET amt0 = (
					SELECT b.amt_sum
					FROM drc20_collect_address b
					WHERE 
						swap_v2_liquidity.tick0_id = b.tick AND 
						swap_v2_liquidity.reserves_address = b.holder_address
				)
				WHERE EXISTS (
					SELECT 1
					FROM drc20_collect_address b
					WHERE 
						swap_v2_liquidity.tick0_id = b.tick AND 
						swap_v2_liquidity.reserves_address = b.holder_address AND 
				)`).Error
	if err != nil {
		return fmt.Errorf("UpdateLiquidity error: %s", err.Error())
	}

	err = tx.Exec(`UPDATE swap_v2_liquidity
				SET amt0 = (
					SELECT b.amt
					FROM meme20_collect_address b
					WHERE 
						swap_v2_liquidity.tick0_id = b.tick_id AND 
						swap_v2_liquidity.reserves_address = b.holder_address
				)
				WHERE EXISTS (
					SELECT 1
					FROM meme20_collect_address b
					WHERE 
						swap_v2_liquidity.tick0_id = b.tick_id AND 
						swap_v2_liquidity.reserves_address = b.holder_address AND 
				)`).Error
	if err != nil {
		return fmt.Errorf("UpdateLiquidity error: %s", err.Error())
	}

	// tick1
	err = tx.Exec(` UPDATE swap_v2_liquidity
				SET amt1 = (
					SELECT b.amt_sum
					FROM drc20_collect_address b
					WHERE 
						swap_v2_liquidity.tick1_id = b.tick AND 
						swap_v2_liquidity.reserves_address = b.holder_address
				)
				WHERE EXISTS (
					SELECT 1
					FROM drc20_collect_address b
					WHERE 
						swap_v2_liquidity.tick1_id = b.tick AND 
						swap_v2_liquidity.reserves_address = b.holder_address AND 
				)`).Error
	if err != nil {
		return err
	}

	err = tx.Exec(` UPDATE swap_v2_liquidity
				SET amt1 = (
					SELECT b.amt
					FROM meme20_collect_address b
					WHERE 
						swap_v2_liquidity.tick1_id = b.tick_id AND 
						swap_v2_liquidity.reserves_address = b.holder_address
				)
				WHERE EXISTS (
					SELECT 1
					FROM meme20_collect_address b
					WHERE 
						swap_v2_liquidity.tick1_id = b.tick_id AND 
						swap_v2_liquidity.reserves_address = b.holder_address AND 
				)`).Error
	if err != nil {
		return err
	}

	// pair
	err = tx.Exec(`UPDATE swap_v2_liquidity
				SET liquidity_total = (
					SELECT b.max_
					FROM meme20_collect b
					WHERE swap_v2_liquidity.pair_id = b.tick_id
				)`).Error
	if err != nil {
		return err
	}

	return nil
}

func (db *DBClient) FindSwapV2PriceAll() ([]*SwapV2Price, int64, error) {

	liquidityAll := make([]*models.SwapV2Liquidity, 0)
	total := int64(0)
	err := db.DB.Where("liquidity_total != '0'").Find(&liquidityAll).Limit(-1).Offset(-1).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	liquidityMap := make(map[string]*big.Float)
	noWdoge := make([]*models.SwapV2Liquidity, 0)
	for _, v := range liquidityAll {
		if v.Tick0Id != "WDOGE(WRAPPED-DOGE)" && v.Tick1Id != "WDOGE(WRAPPED-DOGE)" {
			noWdoge = append(noWdoge, v)
			continue
		}

		if v.Tick0Id == "WDOGE(WRAPPED-DOGE)" {
			liquidityMap[v.Tick1Id] = new(big.Float).Quo(new(big.Float).SetInt(v.Amt0.Int()), new(big.Float).SetInt(v.Amt1.Int()))
		} else {
			liquidityMap[v.Tick0Id] = new(big.Float).Quo(new(big.Float).SetInt(v.Amt1.Int()), new(big.Float).SetInt(v.Amt0.Int()))
		}
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

	liquidityMapNoDoge := make(map[string]*big.Float)
	for _, v := range noWdoge {
		price0, ok0 := liquidityMap[v.Tick0Id]
		price1, ok1 := liquidityMap[v.Tick1Id]
		if ok0 && ok1 {
			continue
		}

		if ok0 {
			price := new(big.Float).Quo(new(big.Float).SetInt(v.Amt0.Int()), new(big.Float).SetInt(v.Amt1.Int()))
			price = new(big.Float).Mul(price, price0)
			liquidityMapNoDoge[v.Tick1Id] = price

		} else if ok1 {

			price := new(big.Float).Quo(new(big.Float).SetInt(v.Amt1.Int()), new(big.Float).SetInt(v.Amt0.Int()))
			price = new(big.Float).Mul(price, price1)
			liquidityMapNoDoge[v.Tick0Id] = price
		}
	}

	for k, v := range liquidityMapNoDoge {

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
