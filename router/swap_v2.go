package router

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"github.com/dogecoinw/doged/rpcclient"
	"github.com/gin-gonic/gin"
	"math/big"
	"net/http"
)

type SwapV2Router struct {
	dbc  *storage.DBClient
	node *rpcclient.Client
}

func NewSwapV2Router(db *storage.DBClient, node *rpcclient.Client) *SwapV2Router {
	return &SwapV2Router{
		dbc:  db,
		node: node,
	}
}

func (r *SwapV2Router) Order(c *gin.Context) {
	params := &struct {
		OrderId       string `json:"order_id"`
		Op            string `json:"op"`
		PairId        string `json:"pair_id"`
		Tick0Id       string `json:"tick0_id"`
		Tick1Id       string `json:"tick1_id"`
		TxHash        string `json:"tx_hash"`
		HolderAddress string `json:"holder_address"`
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
		c.JSON(http.StatusOK, result)
		return
	}

	filter := &models.SwapV2Info{
		OrderId:       params.OrderId,
		Op:            params.Op,
		PairId:        params.PairId,
		Tick0Id:       params.Tick0Id,
		Tick1Id:       params.Tick1Id,
		TxHash:        params.TxHash,
		HolderAddress: params.HolderAddress,
		BlockNumber:   params.BlockNumber,
	}

	infos := make([]*models.SwapV2Info, 0)
	total := int64(0)
	err := r.dbc.DB.Model(&models.SwapV2Info{}).
		Where(filter).
		Count(&total).
		Order("id desc").
		Limit(params.Limit).
		Offset(params.OffSet).
		Find(&infos).Error

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

func (r *SwapV2Router) Liquidity(c *gin.Context) {
	params := &struct {
		PairId  string `json:"pair_id"`
		Tick0Id string `json:"tick0_id"`
		Tick1Id string `json:"tick1_id"`
		Limit   int    `json:"limit"`
		OffSet  int    `json:"offset"`
	}{
		Limit:  -1,
		OffSet: -1,
	}

	if err := c.ShouldBindJSON(&params); err != nil {
		result := &utils.HttpResult{}
		result.Code = 400
		result.Msg = err.Error()
		c.JSON(http.StatusOK, result)
		return
	}

	filter := &models.SwapV2Liquidity{
		PairId:  params.PairId,
		Tick0Id: params.Tick0Id,
		Tick1Id: params.Tick1Id,
	}

	infos := make([]*models.SwapV2Liquidity, 0)
	total := int64(0)
	err := r.dbc.DB.Model(&models.SwapV2Liquidity{}).Where(filter).Count(&total).Limit(params.Limit).Offset(params.OffSet).Find(&infos).Error
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

func (r *SwapV2Router) SwapLiquidityHolder(c *gin.Context) {
	params := &struct {
		PairId        string `json:"pair_id"`
		Tick0Id       string `json:"tick0_id"`
		Tick1Id       string `json:"tick1_id"`
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
		c.JSON(http.StatusOK, result)
		return
	}

	type QueryResult struct {
		Liquidity      *models.Number `gorm:"column:amt" json:"liquidity"`
		LiquidityTotal *models.Number `gorm:"column:liquidity_total" json:"liquidity_total"`
		Reserve0       *models.Number `gorm:"column:amt0" json:"reserve0"`
		Reserve1       *models.Number `gorm:"column:amt1" json:"reserve1"`
		Price          *big.Float     `gorm:"-" json:"price"`
		Tick0          string         `json:"tick0"`
		Tick0Id        string         `json:"tick0_id"`
		Tick1          string         `json:"tick1"`
		Tick1Id        string         `json:"tick1_id"`
		PairId         string         `gorm:"column:pair_id"  json:"pair_id"`
		HolderAddress  string         `gorm:"column:holder_address" json:"holder_address"`
	}

	var results []QueryResult
	var dbModels []*QueryResult
	var total int64

	subQuery := r.dbc.DB.Table("meme20_collect_address dca").
		Select("sl.pair_id, dca.amt, dca.holder_address, sl.liquidity_total, sl.amt0, sl.amt1, sl.tick0, sl.tick0_id, sl.tick1, sl.tick1_id").
		Joins("left join swap_v2_liquidity sl on sl.pair_id = dca.tick_id").
		Where("sl.pair_id != ''")

	if params.HolderAddress != "" {
		subQuery = subQuery.Where("dca.holder_address = ?", params.HolderAddress)
	}

	if params.PairId != "" {
		subQuery = subQuery.Where("sl.pair_id = ?", params.PairId)
	}

	if params.Tick0Id != "" {
		subQuery = subQuery.Where("sl.tick0_id = ?", params.Tick0Id)
	}

	if params.Tick1Id != "" {
		subQuery = subQuery.Where("sl.tick1_id = ?", params.Tick1Id)
	}

	err := subQuery.
		Count(&total).Limit(params.Limit).Offset(params.OffSet).Scan(&results).Error

	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	for _, res := range results {
		dbModels = append(dbModels, &QueryResult{
			HolderAddress:  res.HolderAddress,
			PairId:         res.PairId,
			Tick0:          res.Tick0,
			Tick0Id:        res.Tick0Id,
			Tick1:          res.Tick1,
			Tick1Id:        res.Tick1Id,
			Liquidity:      res.Liquidity,
			LiquidityTotal: res.LiquidityTotal,
			Price:          new(big.Float).Quo(new(big.Float).SetInt(res.Reserve0.Int()), new(big.Float).SetInt(res.Reserve1.Int())),
			Reserve0:       (*models.Number)(new(big.Int).Div(new(big.Int).Mul(res.Reserve0.Int(), res.Liquidity.Int()), res.LiquidityTotal.Int())),
			Reserve1:       (*models.Number)(new(big.Int).Div(new(big.Int).Mul(res.Reserve1.Int(), res.Liquidity.Int()), res.LiquidityTotal.Int())),
		})

	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = dbModels
	result.Total = total

	c.JSON(http.StatusOK, result)
}

func (r *SwapV2Router) SwapPrice(c *gin.Context) {

	result := &utils.HttpResult{}

	pumpPrices, ptotal, err := r.dbc.FindPumpPriceAll()

	if err != nil {
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusBadRequest, result)
		return
	}

	swapPrices, total, err := r.dbc.FindSwapV2PriceAll()
	if err != nil {
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusBadRequest, result)
		return
	}

	result.Code = 200
	result.Msg = "success"
	result.Data = append(pumpPrices, swapPrices...)
	result.Total = ptotal + total

	c.JSON(http.StatusOK, result)
}
