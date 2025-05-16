package router

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"errors"
	"github.com/dogecoinw/doged/rpcclient"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type PumpRouter struct {
	dbc  *storage.DBClient
	node *rpcclient.Client
}

func NewPumpRouter(db *storage.DBClient, node *rpcclient.Client) *PumpRouter {
	return &PumpRouter{
		dbc:  db,
		node: node,
	}
}

func (r *PumpRouter) Order(c *gin.Context) {
	params := &struct {
		OrderId       string `json:"order_id"`
		Op            string `json:"op"`
		TickId        string `json:"tick_id"`
		HolderAddress string `json:"holder_address"`
		TxHash        string `json:"tx_hash"`
		BlockNumber   int64  `json:"block_number"`
		Limit         int    `json:"limit"`
		OffSet        int    `json:"offset"`
	}{
		Limit:  10,
		OffSet: 0,
	}

	if err := c.ShouldBindJSON(&params); err != nil {
		result := &utils.HttpResult{}
		result.Code = 400
		result.Msg = err.Error()
		c.JSON(http.StatusBadRequest, result)
		return
	}

	filter := &models.PumpInfo{
		OrderId:       params.OrderId,
		Op:            params.Op,
		HolderAddress: params.HolderAddress,
		TxHash:        params.TxHash,
		BlockNumber:   params.BlockNumber,
	}

	infos := make([]*models.PumpInfo, 0)
	total := int64(0)
	subQuery := r.dbc.DB.Model(&models.PumpInfo{}).Where(filter)
	if params.TickId != "" {
		subQuery = subQuery.Where("tick0_id = ? or tick1_id = ?", params.TickId, params.TickId)
	}

	err := subQuery.Count(&total).Order("id desc").Limit(params.Limit).Offset(params.OffSet).Find(&infos).Error
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = infos
	result.Total = total

	c.JSON(http.StatusOK, result)

}

func (r *PumpRouter) MergeOrder(c *gin.Context) {

	params := &struct {
		TickId        string `json:"tick_id"`
		HolderAddress string `json:"holder_address"`
		OrderStatus   int    `json:"order_status"`
		BlockNumber   int64  `json:"block_number"`
		Limit         int    `json:"limit"`
		OffSet        int    `json:"offset"`
	}{
		Limit:  10,
		OffSet: 0,
	}

	if err := c.ShouldBindJSON(&params); err != nil {
		result := &utils.HttpResult{}
		result.Code = 400
		result.Msg = err.Error()
		c.JSON(http.StatusBadRequest, result)
		return
	}

	infos := make([]*MergeSwapPumpOrderResult, 0)

	// First query (swap_v2_info)
	swapQuery := r.dbc.DB.Table("swap_v2_info").
		Select("'swap' as p, op, tick0_id, tick0, tick1_id, tick1, amt0, amt1, amt0_out, amt1_out, tx_hash, holder_address, order_status, create_date")

	if params.TickId != "" {
		swapQuery = swapQuery.Where("tick0_id = ? or tick1_id = ?", params.TickId, params.TickId)
	}

	if params.HolderAddress != "" {
		swapQuery = swapQuery.Where("holder_address = ?", params.HolderAddress)
	}

	switch params.OrderStatus {
	case 1:
		swapQuery = swapQuery.Where("order_status = 1")
	case 2:
		swapQuery = swapQuery.Where("order_status = 0")
	}

	// Add block number filter if provided
	if params.BlockNumber > 0 {
		swapQuery = swapQuery.Where("block_number = ?", params.BlockNumber)
	}

	// Second query (pump_info)
	pumpQuery := r.dbc.DB.Table("pump_info").
		Select("'pump' as p, op, tick0_id, tick0, tick1_id, tick1, amt0, amt1, amt0_out, amt1_out, tx_hash, holder_address, order_status, create_date")

	if params.TickId != "" {
		pumpQuery = pumpQuery.Where("tick0_id = ? or tick1_id = ?", params.TickId, params.TickId)
	}

	if params.HolderAddress != "" {
		pumpQuery = pumpQuery.Where("holder_address = ?", params.HolderAddress)
	}

	switch params.OrderStatus {
	case 1:
		pumpQuery = pumpQuery.Where("order_status = 1")
	case 2:
		pumpQuery = pumpQuery.Where("order_status = 0")
	}

	// Add block number filter if provided
	if params.BlockNumber > 0 {
		pumpQuery = pumpQuery.Where("block_number = ?", params.BlockNumber)
	}

	// Combine queries with UNION and apply ordering and limits
	err := r.dbc.DB.Raw("(? UNION ?) ORDER BY create_date DESC limit ? offset ?", swapQuery, pumpQuery, params.Limit, params.OffSet).
		Scan(&infos).Error

	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	total0 := int64(0)
	total1 := int64(0)
	// 两个需要加起来

	subQuery := r.dbc.DB.Table("pump_info")
	if params.TickId != "" {
		subQuery = subQuery.Where("tick0_id = ? or tick1_id = ?", params.TickId, params.TickId)
	}

	if params.HolderAddress != "" {
		subQuery = subQuery.Where("holder_address = ?", params.HolderAddress)
	}

	err = subQuery.Count(&total0).Error
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	subQuery = r.dbc.DB.Table("swap_v2_info")

	if params.TickId != "" {
		subQuery = subQuery.Where("tick0_id = ? or tick1_id = ?", params.TickId, params.TickId)
	}

	if params.HolderAddress != "" {
		subQuery = subQuery.Where("holder_address = ?", params.HolderAddress)
	}

	err = subQuery.Count(&total1).Error
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = infos
	result.Total = total0 + total1

	c.JSON(http.StatusOK, result)

}

func (r *PumpRouter) Liquidity(c *gin.Context) {
	params := &struct {
		HolderAddress string `json:"holder_address"`
		TickId        string `json:"tick_id"`
		Limit         int    `json:"limit"`
		OffSet        int    `json:"offset"`
	}{
		Limit:  10,
		OffSet: 0,
	}

	if err := c.ShouldBindJSON(&params); err != nil {
		result := &utils.HttpResult{}
		result.Code = 400
		result.Msg = err.Error()
		c.JSON(http.StatusBadRequest, result)
		return
	}

	filter := &models.PumpLiquidity{
		HolderAddress: params.HolderAddress,
		Tick0Id:       params.TickId,
	}

	infos := make([]*models.PumpLiquidity, 0)
	total := int64(0)
	err := r.dbc.DB.Model(&models.PumpLiquidity{}).Where(filter).Count(&total).Order("id desc").Limit(params.Limit).Offset(params.OffSet).Find(&infos).Error
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = infos
	result.Total = total

	c.JSON(http.StatusOK, result)
}

// board
func (r *PumpRouter) Board(c *gin.Context) {
	params := &struct {
		Sort      string `json:"sort"`
		SortBy    string `json:"sort_by"`
		SearchKey string `json:"search_key"`
		Limit     int    `json:"limit"`
		OffSet    int    `json:"offset"`
	}{
		Limit:  10,
		OffSet: 0,
	}

	if err := c.ShouldBindJSON(&params); err != nil {
		result := &utils.HttpResult{}
		result.Code = 400
		result.Msg = err.Error()
		c.JSON(http.StatusBadRequest, result)
		return
	}

	if params.Limit > 100 {
		params.Limit = 100
	}

	//OrderBy
	// 1. 交易量
	// 2. 时间排序
	// 3. 价格排序
	infos := make([]*PumpBoard, 0)
	total := int64(0)
	startDate := time.Now()
	timeStamp := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location()).Unix()

	subQuery := r.dbc.DB.Table("pump_liquidity as pl").
		Select("mc.tick, mc.tick_id, mc.logo, mc.reserve, mc.tag, mc.twitter, mc.telegram, mc.discord, mc.website, mc.youtube, mc.tiktok, mc.name, mc.description, mc.holder_address, mc.transactions, svs.price_change, svs.base_volume, pl.amt0, pl.amt1, pl.amt0/pl.amt1 as price, pl.holder_address, pl.king_date, pl.create_date, sl.amt0 as swap_amt0, sl.amt1 as swap_amt1,  uca.profile_photo, uca.user_name, uca.bio, "+
			"(SELECT COUNT(id) FROM tg_bot.user_chat WHERE tg_bot.user_chat.tick_id = mc.tick_id) AS replies, "+
			"(SELECT COUNT(id) FROM meme20_collect_address WHERE tick_id = pl.tick0_id and amt != '0') AS holders").
		Joins("left join meme20_collect as mc on pl.tick0_id = mc.tick_id").
		Joins("left join swap_v2_liquidity as sl on pl.tick0_id = sl.tick0_id and sl.tick1_id = 'WDOGE(WRAPPED-DOGE)'").
		Joins("left join (SELECT * from tg_bot.account where id in (SELECT max(id) from tg_bot.account group by address)) as uca on uca.address = mc.holder_address").
		Joins("left join (WITH RankedRecords AS (SELECT tick_id, COALESCE(((close_price - open_price) / open_price) * 100, 0) AS price_change, base_volume, date_interval, ROW_NUMBER() OVER (PARTITION BY tick_id ORDER BY id DESC) as rn FROM swap_v2_summary where date_interval = '1d' and last_date = ?) SELECT * FROM RankedRecords WHERE rn = 1) as svs on svs.tick_id = pl.tick0_id", time.Unix(timeStamp, 0).Format("2006-01-02 15:04:05"))

	if params.SearchKey != "" {
		subQuery = subQuery.Where("mc.name like ? or mc.tick_id like ? or mc.tick like ?", "%"+params.SearchKey+"%", "%"+params.SearchKey+"%", "%"+params.SearchKey+"%")
	}

	if params.Sort == "feat" {
		if params.SortBy == "desc" {
			subQuery = subQuery.Order("mc.transactions desc")
		} else {
			subQuery = subQuery.Order("mc.transactions asc")
		}
	}

	if params.Sort == "create" {
		if params.SortBy == "desc" {
			subQuery = subQuery.Order("pl.create_date desc")
		} else {
			subQuery = subQuery.Order("pl.create_date asc")
		}
	}

	if params.Sort == "price" {
		if params.SortBy == "desc" {
			subQuery = subQuery.Order("price desc")
		} else {
			subQuery = subQuery.Order("price asc")
		}
	}

	if params.Sort == "price_change" {
		if params.SortBy == "desc" {
			subQuery = subQuery.Order("svs.price_change desc")
		} else {
			subQuery = subQuery.Order("svs.price_change asc")
		}
	}

	subQuery.Count(&total).Limit(params.Limit).
		Offset(params.OffSet)

	err := subQuery.Scan(&infos).Error
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = infos
	result.Total = total

	c.JSON(http.StatusOK, result)
}

func (r *PumpRouter) K(c *gin.Context) {
	type params struct {
		TickId       string `json:"tick_id"`
		DateInterval string `json:"date_interval"`
		From         int64  `json:"from"`
		To           int64  `json:"to"`
	}

	p := &params{
		DateInterval: "1d",
	}

	if err := c.ShouldBindJSON(&p); err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusOK, result)
		return
	}

	sec := utils.IntervalToSecond(p.DateInterval)

	if p.To-p.From < 0 || p.To-p.From > 1000*sec {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "Incorrect time interval"
		c.JSON(http.StatusOK, result)
		return
	}

	results := make([]models.Summary, 0)
	err := r.dbc.DB.Table("swap_v2_summary").Where("tick_id = ? and date_interval = ? and time_stamp >= ? and time_stamp <= ?", p.TickId, p.DateInterval, p.From, p.To).
		Find(&results).Error
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	// 获取最新区块的时间戳
	//blockCount, _ := r.node.GetBlockCount()
	//block, _ := r.node.GetBlockHash(blockCount)
	//blockInfo, _ := r.node.GetBlockVerboseBool(block)
	tnow := time.Now().Unix()

	// 如果 results 是0 个， 那么获取最后一条
	if len(results) == 0 {
		summ := &models.Summary{}
		err = r.dbc.DB.Table("swap_v2_summary").Where("tick_id = ? and date_interval = ? and time_stamp <= ?", p.TickId, p.DateInterval, p.From).Order("id desc").First(&summ).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = r.dbc.DB.Table("swap_v2_summary").Where("tick_id = ? and date_interval = ? ", p.TickId, p.DateInterval).Order("id desc").First(&summ).Error
				if err != nil {
					result := &utils.HttpResult{}
					result.Code = 500
					result.Msg = err.Error()
					c.JSON(http.StatusInternalServerError, result)
					return
				}
			} else {
				result := &utils.HttpResult{}
				result.Code = 500
				result.Msg = err.Error()
				c.JSON(http.StatusInternalServerError, result)
				return
			}
		}

		time_stamp := summ.TimeStamp
		temp := 0

		results1 := make([]models.Summary, 0)
		summ0 := *summ
		for summ.TimeStamp > 0 && time_stamp < p.To && time_stamp < tnow {
			if temp == 0 || time_stamp < p.From {
				temp = 1
				time_stamp += sec
				continue
			}

			summ0.OpenPrice = summ0.ClosePrice
			summ0.HighestBid = summ0.ClosePrice
			summ0.LowestAsk = summ0.ClosePrice

			summ0.TimeStamp = time_stamp
			summ0.LastDate = time.Unix(time_stamp, 0).Format("2006-01-02 15:04:05")
			summ0.BaseVolume = models.NewNumber(0)
			results1 = append(results1, summ0)
			time_stamp += sec
		}

		result := &utils.HttpResult{}
		result.Code = 200
		result.Msg = "success"
		result.Data = results1
		result.Total = int64(len(results1))
		c.JSON(http.StatusOK, result)
		return
	}

	// 根据时间间隔没有的补上
	results1 := make([]models.Summary, 0)
	time_stamp := int64(0)
	summ0 := models.Summary{}

	if len(results) > 0 {
		summ0 = results[0]
		if summ0.TimeStamp > p.From {
			summ := &models.Summary{}
			err = r.dbc.DB.Table("swap_v2_summary").Where("tick_id = ? and date_interval = ? and time_stamp < ?", p.TickId, p.DateInterval, p.From).Order("id desc").First(&summ).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					err = r.dbc.DB.Table("swap_v2_summary").Where("tick_id = ? and date_interval = ? ", p.TickId, p.DateInterval).Order("id desc").First(&summ).Error
					if err != nil {
						result := &utils.HttpResult{}
						result.Code = 500
						result.Msg = err.Error()
						c.JSON(http.StatusInternalServerError, result)
						return
					}
				} else {
					result := &utils.HttpResult{}
					result.Code = 500
					result.Msg = err.Error()
					c.JSON(http.StatusInternalServerError, result)
					return
				}
			}

			time_stamp1 := summ.TimeStamp
			summ1 := *summ

			for summ.TimeStamp > 0 && time_stamp1 < summ0.TimeStamp {
				if time_stamp1 < p.From {
					time_stamp1 += sec
					continue
				}

				summ1.OpenPrice = summ1.ClosePrice
				summ1.HighestBid = summ1.ClosePrice
				summ1.LowestAsk = summ1.ClosePrice

				summ1.TimeStamp = time_stamp1
				summ1.LastDate = time.Unix(time_stamp1, 0).Format("2006-01-02 15:04:05")
				summ1.BaseVolume = models.NewNumber(0)
				results1 = append(results1, summ1)
				time_stamp1 += sec
			}
		}
	}

	for i, summ := range results {

		if i == 0 {
			time_stamp = summ.TimeStamp
			results1 = append(results1, summ)
			continue
		}

		time_stamp += sec

		for summ.TimeStamp != time_stamp {
			if time_stamp > p.To {
				break
			}

			summ1 := models.Summary{}
			summ1.ClosePrice = summ0.ClosePrice
			summ1.OpenPrice = summ0.ClosePrice
			summ1.HighestBid = summ0.ClosePrice
			summ1.LowestAsk = summ0.ClosePrice

			summ1.TimeStamp = time_stamp
			summ1.LastDate = time.Unix(time_stamp, 0).Format("2006-01-02 15:04:05")
			summ1.BaseVolume = models.NewNumber(0)
			results1 = append(results1, summ1)
			time_stamp += sec
		}

		if time_stamp > p.To {
			break
		}

		summ.LastDate = time.Unix(time_stamp, 0).Format("2006-01-02 15:04:05")
		results1 = append(results1, summ)
		summ0 = summ
	}

	temp := 0
	for len(results) > 0 && time_stamp < p.To && time_stamp < tnow {
		if temp == 0 {
			temp = 1
			time_stamp += sec
			continue
		}

		summ0.OpenPrice = summ0.ClosePrice
		summ0.HighestBid = summ0.ClosePrice
		summ0.LowestAsk = summ0.ClosePrice

		summ0.TimeStamp = time_stamp
		summ0.LastDate = time.Unix(time_stamp, 0).Format("2006-01-02 15:04:05")
		summ0.BaseVolume = models.NewNumber(0)
		results1 = append(results1, summ0)
		time_stamp += sec
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = results1
	result.Total = int64(len(results1))
	c.JSON(http.StatusOK, result)
}

func (r *PumpRouter) King(c *gin.Context) {

	params := &struct {
		Limit  int `json:"limit"`
		OffSet int `json:"offset"`
	}{
		Limit:  10,
		OffSet: 0,
	}

	if err := c.ShouldBindJSON(&params); err != nil {
		result := &utils.HttpResult{}
		result.Code = 400
		result.Msg = err.Error()
		c.JSON(http.StatusBadRequest, result)
		return
	}

	type King struct {
		TickId        string           `json:"tick_id"`
		Tick          string           `json:"tick"`
		Logo          string           `json:"logo"`
		Name          string           `json:"name"`
		Amt0          *models.Number   `json:"amt0"`
		Amt1          *models.Number   `json:"amt1"`
		SwapAmt0      *models.Number   `json:"swap_amt0"`
		SwapAmt1      *models.Number   `json:"swap_amt1"`
		HolderAddress string           `json:"holder_address"`
		Transactions  int64            `json:"transactions"`
		Replies       int64            `json:"replies"`
		KingDate      models.LocalTime `json:"king_date"`
		UpdateDate    models.LocalTime `json:"update_date"`
		CreateDate    models.LocalTime `json:"create_date"`
	}

	king := make([]King, 0)
	err := r.dbc.DB.Table("pump_liquidity as pl").
		Select("mc.tick_id, mc.tick, mc.logo, mc.name, mc.holder_address, mc.transactions, pl.king_date, pl.update_date, pl.create_date, pl.amt0, pl.amt1, sl.amt0 as swap_amt0, sl.amt1 as swap_amt1," +
			"(SELECT COUNT(id) FROM tg_bot.user_chat WHERE tg_bot.user_chat.tick_id = mc.tick_id) AS replies").
		Joins("left join meme20_collect as mc on pl.tick0_id = mc.tick_id").
		Joins("left join swap_v2_liquidity as sl on pl.tick0_id = sl.tick0_id and sl.tick1_id = 'WDOGE(WRAPPED-DOGE)'").
		Order("pl.king_date desc").Limit(params.Limit).Offset(params.OffSet).Find(&king).Error

	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = king
	c.JSON(http.StatusOK, result)
}
