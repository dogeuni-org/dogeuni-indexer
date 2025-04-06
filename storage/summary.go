package storage

import (
	"dogeuni-indexer/models"
	"errors"
	"github.com/dogecoinw/go-dogecoin/log"
	"gorm.io/gorm"
	"math/big"
	"time"
)

func (db *DBClient) SummarySwapV2(swap *models.SwapV2Info) error {
	tx := db.DB.Begin()
	price := 0.0
	volume := big.NewInt(0)
	tickId := swap.Tick0Id
	if swap.Tick0Id == "WDOGE(WRAPPED-DOGE)" {
		volume = swap.Amt0.Int()
		priceF := new(big.Float).Quo(new(big.Float).SetInt(swap.Amt0.Int()), new(big.Float).SetInt(swap.Amt1Out.Int()))
		price, _ = priceF.Float64()
		tickId = swap.Tick1Id
	} else if swap.Tick1Id == "WDOGE(WRAPPED-DOGE)" {
		volume = swap.Amt1Out.Int()
		priceF := new(big.Float).Quo(new(big.Float).SetInt(swap.Amt1Out.Int()), new(big.Float).SetInt(swap.Amt0.Int()))
		price, _ = priceF.Float64()
	}

	startDate := time.Now()
	timeStamp := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute(), 0, 0, startDate.Location()).Unix()

	err := SummaryK(tx, tickId, price, volume, timeStamp, "1m")
	if err != nil {
		return err
	}

	temp := startDate.Minute() % 5
	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute()-temp, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "5m")
	if err != nil {
		return err
	}

	temp = startDate.Minute() % 15
	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute()-temp, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "15m")
	if err != nil {
		return err
	}

	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "1h")
	if err != nil {
		return err
	}

	temp = startDate.Hour() % 4
	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour()-temp, 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "4h")
	if err != nil {
		return err
	}

	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "1d")
	if err != nil {
		return err
	}

	if startDate.Weekday() == time.Sunday {
		timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day()-6, 0, 0, 0, 0, startDate.Location()).Unix()
	} else {
		timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day()-int(startDate.Weekday())+1, 0, 0, 0, 0, startDate.Location()).Unix()
	}

	err = SummaryK(tx, tickId, price, volume, timeStamp, "1w")
	if err != nil {
		return err
	}

	timeStamp = time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "1M")
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		log.Error("SummarySwapV2 tx.Commit err: %s", err.Error())
		tx.Rollback()
	}

	return nil
}

func (db *DBClient) SummaryPump(pump *models.PumpInfo) error {
	tx := db.DB.Begin()
	price := 0.0
	volume := big.NewInt(0)
	tickId := pump.Tick0Id
	if pump.Tick0Id == "WDOGE(WRAPPED-DOGE)" {
		volume = pump.Amt0.Int()
		priceF := new(big.Float).Quo(new(big.Float).SetInt(pump.Amt0.Int()), new(big.Float).SetInt(pump.Amt1.Int()))
		price, _ = priceF.Float64()
		tickId = pump.Tick1Id
	} else if pump.Tick1Id == "WDOGE(WRAPPED-DOGE)" {
		volume = pump.Amt1.Int()
		priceF := new(big.Float).Quo(new(big.Float).SetInt(pump.Amt1.Int()), new(big.Float).SetInt(pump.Amt0.Int()))
		price, _ = priceF.Float64()
	}

	startDate := time.Now()
	timeStamp := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute(), 0, 0, startDate.Location()).Unix()

	err := SummaryK(tx, tickId, price, volume, timeStamp, "1m")
	if err != nil {
		return err
	}

	temp := startDate.Minute() % 5
	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute()-temp, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "5m")
	if err != nil {
		return err
	}

	temp = startDate.Minute() % 15
	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute()-temp, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "15m")
	if err != nil {
		return err
	}

	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "1h")
	if err != nil {
		return err
	}

	temp = startDate.Hour() % 4
	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour()-temp, 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "4h")
	if err != nil {
		return err
	}

	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "1d")
	if err != nil {
		return err
	}

	if startDate.Weekday() == time.Sunday {
		timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day()-6, 0, 0, 0, 0, startDate.Location()).Unix()
	} else {
		timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day()-int(startDate.Weekday())+1, 0, 0, 0, 0, startDate.Location()).Unix()
	}

	err = SummaryK(tx, tickId, price, volume, timeStamp, "1w")
	if err != nil {
		return err
	}

	timeStamp = time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "1M")
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		log.Error("SummaryPump tx.Commit err: %s", err.Error())
		tx.Rollback()
	}

	return nil
}

func (db *DBClient) SummaryPumpCreate(pump *models.PumpInfo) error {
	tx := db.DB.Begin()
	price := 0.0
	volume := big.NewInt(0)
	tickId := pump.Tick0Id
	if pump.Tick0Id == "WDOGE(WRAPPED-DOGE)" {
		priceF := new(big.Float).Quo(new(big.Float).SetInt(pump.Amt0Out.Int()), new(big.Float).SetInt(pump.Amt1Out.Int()))
		price, _ = priceF.Float64()
		tickId = pump.Tick1Id
	} else if pump.Tick1Id == "WDOGE(WRAPPED-DOGE)" {
		priceF := new(big.Float).Quo(new(big.Float).SetInt(pump.Amt1Out.Int()), new(big.Float).SetInt(pump.Amt0Out.Int()))
		price, _ = priceF.Float64()
	}

	startDate := time.Now()
	timeStamp := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute(), 0, 0, startDate.Location()).Unix()

	err := SummaryK(tx, tickId, price, volume, timeStamp, "1m")
	if err != nil {
		return err
	}

	temp := startDate.Minute() % 5
	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute()-temp, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "5m")
	if err != nil {
		return err
	}

	temp = startDate.Minute() % 15
	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute()-temp, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "15m")
	if err != nil {
		return err
	}

	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "1h")
	if err != nil {
		return err
	}

	temp = startDate.Hour() % 4
	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour()-temp, 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "4h")
	if err != nil {
		return err
	}

	timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "1d")
	if err != nil {
		return err
	}

	if startDate.Weekday() == time.Sunday {
		timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day()-6, 0, 0, 0, 0, startDate.Location()).Unix()
	} else {
		timeStamp = time.Date(startDate.Year(), startDate.Month(), startDate.Day()-int(startDate.Weekday())+1, 0, 0, 0, 0, startDate.Location()).Unix()
	}

	err = SummaryK(tx, tickId, price, volume, timeStamp, "1w")
	if err != nil {
		return err
	}

	timeStamp = time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, startDate.Location()).Unix()
	err = SummaryK(tx, tickId, price, volume, timeStamp, "1M")
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		log.Error("SummaryPump tx.Commit err: %s", err.Error())
		tx.Rollback()
	}

	return nil
}

func SummaryK(tx *gorm.DB, tickId string, price float64, volume *big.Int, timeStamp int64, dateInterval string) error {
	summary := &models.Summary{}

	err := tx.Table("swap_v2_summary").Where("tick_id = ? and time_stamp = ? and date_interval = ?", tickId, timeStamp, dateInterval).First(summary).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if len(summary.TickId) == 0 {
		summarya := &models.Summary{}
		err := tx.Table("swap_v2_summary").Where("tick_id = ? and date_interval = ?", tickId, dateInterval).Order("id desc").First(summarya).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if len(summarya.TickId) != 0 {
			summary.OpenPrice = summarya.ClosePrice
		} else {
			summary.OpenPrice = price
		}

		summary.TickId = tickId
		summary.ClosePrice = price
		summary.LowestAsk = price
		summary.HighestBid = price
		summary.TimeStamp = timeStamp
		summary.LastDate = time.Unix(timeStamp, 0).Format("2006-01-02 15:04:05")
		summary.DateInterval = dateInterval
		summary.BaseVolume = (*models.Number)(volume)

		err = tx.Table("swap_v2_summary").Create(summary).Error
		if err != nil {
			return err
		}

	} else {
		summary.BaseVolume = (*models.Number)(new(big.Int).Add(summary.BaseVolume.Int(), volume))
		summary.ClosePrice = price
		if price > summary.HighestBid {
			summary.HighestBid = price
		}

		if price < summary.LowestAsk {
			summary.LowestAsk = price
		}

		updates := map[string]interface{}{
			"close_price": summary.ClosePrice,
			"lowest_ask":  summary.LowestAsk,
			"highest_bid": summary.HighestBid,
			"base_volume": summary.BaseVolume,
		}

		err = tx.Table("swap_v2_summary").Where("tick_id = ? and time_stamp = ? and date_interval = ?", tickId, timeStamp, dateInterval).Updates(updates).Error
		if err != nil {
			return err
		}
	}

	return nil
}
