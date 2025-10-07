package router

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"github.com/dogecoinw/doged/rpcclient"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Meme20Router struct {
	dbc   *storage.DBClient
	node  *rpcclient.Client
	level *storage.LevelDB
}

func NewMeme20Router(db *storage.DBClient, node *rpcclient.Client, level *storage.LevelDB) *Meme20Router {
	return &Meme20Router{
		dbc:   db,
		node:  node,
		level: level,
	}
}

func (r *Meme20Router) Order(c *gin.Context) {
	params := &struct {
		OrderId       string `json:"order_id"`
		Op            string `json:"op"`
		TickId        string `json:"tick_id"`
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

	filter := &models.Meme20Info{
		OrderId:       params.OrderId,
		Op:            params.Op,
		TickId:        params.TickId,
		HolderAddress: params.HolderAddress,
		ToAddress:     params.ToAddress,
		TxHash:        params.TxHash,
		BlockNumber:   params.BlockNumber,
	}

	infos := make([]*models.Meme20Info, 0)
	total := int64(0)

	subQuery := r.dbc.DB.Model(&models.Meme20Info{})

	if params.Address != "" {
		filter.HolderAddress = ""
		filter.ToAddress = ""
		subQuery = subQuery.Where("holder_address = ? OR to_address = ? ", params.Address, params.Address)
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

func (r *Meme20Router) History(c *gin.Context) {
	params := &struct {
		TickId  string `json:"tick_id"`
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

	filter := &models.Meme20Revert{
		TickId: params.TickId,
	}

	infos := make([]*models.Meme20Revert, 0)
	total := int64(0)

	subQuery := r.dbc.DB.Table("meme20_revert as me").Select("me.*, mc.tick, mc.name").
		Joins("LEFT JOIN meme20_collect AS mc ON mc.tick_id = me.tick_id")

	if params.Address != "" {
		subQuery = subQuery.Where("me.from_address = ? OR me.to_address = ? ", params.Address, params.Address)
	}

	if params.TickId != "" {
		subQuery = subQuery.Where("me.tick_id = ?", params.TickId)
	}

	err := subQuery.Where(filter).Count(&total).Order("me.id desc").Limit(params.Limit).Offset(params.OffSet).Find(&infos).Error
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

// Query all NFTs under an address
func (r *Meme20Router) CollectAddress(c *gin.Context) {
	params := &struct {
		TickId        string `json:"tick_id"`
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

	results := make([]*models.Meme20CollectAddress, 0)
	subQuery := r.dbc.DB.Table("meme20_collect_address AS mca").
		Select(`mca.tick_id, mca.amt, mc.max_, mc.name, mc.tick, mc.logo, svl.amt0 as lp_amt0 , svl.amt1 as lp_amt1,
			mca.transactions, 
			mca.holder_address,
	        mca.update_date, 
			mca.create_date`).
		Joins("LEFT JOIN meme20_collect AS mc ON mca.tick_id = mc.tick_id").
		Joins("LEFT JOIN swap_v2_liquidity AS svl ON svl.pair_id = mca.tick_id")

	if params.TickId != "" {
		subQuery = subQuery.Where("mca.tick_id = ?", params.TickId)
	}

	if params.HolderAddress != "" {
		subQuery = subQuery.Where("mca.holder_address = ?", params.HolderAddress)
	}

	total := int64(0)
	err := subQuery.
		Where("mca.amt != '0'").
		Count(&total).
		Order("CAST(mca.amt AS DECIMAL(64,0)) DESC").
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

func (r *Meme20Router) Collect(c *gin.Context) {
	params := &struct {
		HolderAddress string `json:"holder_address"`
		TickId        string `json:"tick_id"`
		SearchKey     string `json:"search_key"`
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

	results := make([]*models.Meme20Collect, 0)
	subQuery := r.dbc.DB.Table("meme20_collect AS di").
		Select(`di.tick, di.tick_id, di.max_, di.dec_, di.name, 
			di.transactions, 
			di.holder_address,
	        di.update_date, 
			di.create_date,
			(select count(id) from meme20_collect_address as mca where mca.tick_id = di.tick_id and mca.amt != '0') AS holders,
            di.logo, di.reserve, di.tag, di.description, di.twitter, di.telegram, di.discord, di.website, di.youtube, di.tiktok, di.is_check`)

	if params.TickId != "" {
		subQuery = subQuery.Where("di.tick_id = ?", params.TickId)
	}

	if params.SearchKey != "" {
		subQuery = subQuery.Where("di.name like ? or di.tick_id like ? or di.tick like ?", "%"+params.SearchKey+"%", "%"+params.SearchKey+"%", "%"+params.SearchKey+"%")
	}

	if params.HolderAddress != "" {
		subQuery = subQuery.Where("di.holder_address = ?", params.HolderAddress)
	}

	total := int64(0)
	err := subQuery.
		Count(&total).
		Order("di.create_date DESC").
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

	if params.HolderAddress == "" {
		for _, result := range results {
			if result.IsCheck == 1 {
				de := ""
				result.Description = &de
				result.Logo = ""
				result.Telegram = nil
				result.Twitter = nil
				result.Discord = nil
				result.Website = nil
				result.Youtube = nil
				result.Tiktok = nil
			}
		}
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = results
	result.Total = total

	c.JSON(http.StatusOK, result)
}
