package router

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"github.com/dogecoinw/doged/rpcclient"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
	"net/http"
)

var (
	cacheDrc20CollectAll *models.Drc20CollectCache
)

type Drc20Router struct {
	dbc   *storage.DBClient
	node  *rpcclient.Client
	ipfs  *shell.Shell
	level *storage.LevelDB
}

func NewDrc20Router(db *storage.DBClient, node *rpcclient.Client, level *storage.LevelDB, ipfs *shell.Shell) *Drc20Router {
	return &Drc20Router{
		dbc:   db,
		node:  node,
		level: level,
		ipfs:  ipfs,
	}
}

func (r *Drc20Router) Order(c *gin.Context) {
	params := &struct {
		OrderId       string `json:"order_id"`
		Op            string `json:"op"`
		Tick          string `json:"tick"`
		HolderAddress string `json:"holder_address"`
		ToAddress     string `json:"to_address"`
		Address       string `json:"address"`
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

	filter := &models.Drc20Info{
		OrderId:       params.OrderId,
		Op:            params.Op,
		Tick:          params.Tick,
		HolderAddress: params.HolderAddress,
		ToAddress:     params.ToAddress,
		TxHash:        params.TxHash,
		BlockNumber:   params.BlockNumber,
	}

	infos := make([]*models.Drc20Info, 0)
	total := int64(0)

	subQuery := r.dbc.DB.Model(&models.Drc20Info{})

	if params.Address != "" {
		filter.HolderAddress = ""
		filter.ToAddress = ""
		subQuery = subQuery.Where("length(to_address) =  34 and (holder_address = ? OR to_address = ?) ", params.Address, params.Address)
	}

	err := subQuery.Where(filter).Count(&total).Order("id desc").Limit(params.Limit).Offset(params.OffSet).Find(&infos).Error
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

func (r *Drc20Router) History(c *gin.Context) {
	params := &struct {
		Tick    string `json:"tick"`
		Address string `json:"address"`
		Limit   int    `json:"limit"`
		OffSet  int    `json:"offset"`
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

	filter := &models.Drc20Revert{
		Tick: params.Tick,
	}

	infos := make([]*models.Drc20Revert, 0)
	total := int64(0)

	subQuery := r.dbc.DB.Model(&models.Drc20Revert{})

	if params.Address != "" {
		subQuery = subQuery.Where("from_address = ? OR to_address = ?", params.Address, params.Address)
	}

	err := subQuery.Where(filter).Count(&total).Order("id desc").Limit(params.Limit).Offset(params.OffSet).Find(&infos).Error
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

func (r *Drc20Router) CollectAddress(c *gin.Context) {
	params := &struct {
		Tick          string `json:"tick"`
		HolderAddress string `json:"holder_address"`
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

	var total int64
	results := make([]*models.Drc20CollectAddress, 0)

	subQuery := r.dbc.DB.Table("drc20_collect_address AS dca").
		Select(`dca.tick, dca.amt_sum, dca.tick, dc.max_, dc.logo,
			dca.transactions, 
			dca.holder_address,
	        dca.update_date, 
			dca.create_date`).
		Joins("LEFT JOIN drc20_collect AS dc ON dca.tick = dc.tick")

	subQuery.Where("dca.amt_sum != '0'")

	if params.Tick != "" {
		subQuery = subQuery.Where("dca.tick = ?", params.Tick)
	}

	if params.HolderAddress != "" {
		subQuery = subQuery.Where("dca.holder_address = ?", params.HolderAddress)
	}

	err := subQuery.
		Count(&total).
		Order("CAST(dca.amt_sum AS DECIMAL(64,0)) DESC").
		Limit(params.Limit).
		Offset(params.OffSet).
		Scan(&results).Error

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
	result.Data = results
	result.Total = total

	c.JSON(http.StatusOK, result)
}

func (r *Drc20Router) Collect(c *gin.Context) {
	params := &struct {
		HolderAddress string `json:"holder_address"`
		Tick          string `json:"tick"`
	}{}

	if err := c.ShouldBindJSON(&params); err != nil {
		result := &utils.HttpResult{}
		result.Code = 400
		result.Msg = err.Error()
		c.JSON(http.StatusBadRequest, result)
		return
	}

	maxHeight := 0
	err := r.dbc.DB.Model(&models.Block{}).Select("max(block_number)").Scan(&maxHeight).Error
	if params.Tick == "" && params.HolderAddress == "" {
		if cacheDrc20CollectAll != nil && cacheDrc20CollectAll.CacheNumber == int64(maxHeight) {
			result := &utils.HttpResult{}
			result.Code = 200
			result.Msg = "success"
			result.Data = cacheDrc20CollectAll.Results
			result.Total = cacheDrc20CollectAll.Total
			c.JSON(http.StatusOK, result)
			return
		}
	}

	results := make([]*models.Drc20CollectRouter, 0)
	subQuery := r.dbc.DB.Table("drc20_collect AS di").
		Select(`di.tick, di.amt_sum as mint_amt, di.max_ as max_amt, di.lim_, di.transactions, di.holder_address as deploy_by,
	        di.update_date AS last_mint_time, (select count(*) from drc20_collect_address where tick = di.tick and amt_sum != '0') AS holders,
			di.create_date AS deploy_time, di.tx_hash as inscription, di.logo, di.introduction, di.white_paper, di.official, di.telegram, di.discorad, di.twitter, di.facebook, di.github,di.is_check`)

	if params.Tick != "" {
		subQuery = subQuery.Where("di.tick = ?", params.Tick)
	}

	if params.HolderAddress != "" {
		subQuery = subQuery.Where("di.holder_address = ?", params.HolderAddress)
	}

	total := int64(0)
	err = subQuery.
		Count(&total).
		Order("di.create_date DESC").
		Scan(&results).Error

	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	if params.HolderAddress == "" {
		for _, result := range results {
			if result.IsCheck == 0 {
				de := ""
				result.Logo = &de
				result.Introduction = &de
				result.WhitePaper = &de
				result.Official = &de
				result.Telegram = &de
				result.Discorad = &de
				result.Twitter = &de
				result.Facebook = &de
				result.Github = &de
			}
		}
	}

	cacheDrc20CollectAll = &models.Drc20CollectCache{
		CacheNumber: int64(maxHeight),
		Results:     results,
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = results
	result.Total = total

	c.JSON(http.StatusOK, result)
}
