package router

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"github.com/dogecoinw/doged/rpcclient"
	"github.com/gin-gonic/gin"
	"net/http"
)

type InviteRouter struct {
	dbc   *storage.DBClient
	node  *rpcclient.Client
	level *storage.LevelDB
}

func NewInviteRouter(db *storage.DBClient, node *rpcclient.Client, level *storage.LevelDB) *InviteRouter {
	return &InviteRouter{
		dbc:   db,
		node:  node,
		level: level,
	}
}

func (r *InviteRouter) Order(c *gin.Context) {
	params := &struct {
		OrderId       string `json:"order_id"`
		Op            string `json:"op"`
		HolderAddress string `json:"holder_address"`
		InviteAddress string `json:"invite_address"`
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

	filter := &models.InviteInfo{
		OrderId:       params.OrderId,
		Op:            params.Op,
		HolderAddress: params.HolderAddress,
		InviteAddress: params.InviteAddress,
		TxHash:        params.TxHash,
		BlockNumber:   params.BlockNumber,
	}

	infos := make([]*models.InviteInfo, 0)
	total := int64(0)

	subQuery := r.dbc.DB.Model(&models.InviteInfo{})

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

func (r *InviteRouter) Collect(c *gin.Context) {
	params := &struct {
		HolderAddress string `json:"holder_address"`
		InviteAddress string `json:"invite_address"`
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

	results := make([]*models.InviteCollect, 0)
	total := int64(0)

	filter := &models.InviteCollect{
		HolderAddress: params.HolderAddress,
		InviteAddress: params.InviteAddress,
	}

	err := r.dbc.DB.Model(&models.InviteCollect{}).Where(filter).Count(&total).Order("id desc").
		Limit(params.Limit).Offset(params.OffSet).
		Find(&results).Error
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

func (r *InviteRouter) PumpReward(c *gin.Context) {
	params := &struct {
		HolderAddress string `json:"holder_address"`
		InviteAddress string `json:"invite_address"`
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

	results := make([]*models.PumpInviteReward, 0)
	total := int64(0)

	subQuery := r.dbc.DB.Table("invite_collect as ic").
		Select("ic.*, COALESCE(ii.invite_reward, 0) as invite_reward").
		Joins("left join pump_invite_reward as ii on ic.invite_address = ii.invite_address and ic.holder_address = ii.holder_address")

	if params.HolderAddress != "" {
		subQuery = subQuery.Where("ic.holder_address = ?", params.HolderAddress)
	}

	if params.InviteAddress != "" {
		subQuery = subQuery.Where("ic.invite_address = ?", params.InviteAddress)
	}

	err := subQuery.Count(&total).Limit(params.Limit).Offset(params.OffSet).Find(&results).Error
	if err != nil {
		println(err.Error())
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

func (r *InviteRouter) PumpRewardTotal(c *gin.Context) {
	params := &struct {
		InviteAddress string `json:"invite_address"`
	}{}

	if err := c.ShouldBindJSON(&params); err != nil {
		result := &utils.HttpResult{}
		result.Code = 400
		result.Msg = err.Error()
		c.JSON(http.StatusBadRequest, result)
		return
	}

	type Total struct {
		RewardTotal  float64 `json:"reward_total"`
		AddressTotal int     `json:"address_total"`
	}

	total := &Total{}
	subQuery := r.dbc.DB.Table("invite_collect as ic").
		Select("sum(COALESCE(ii.invite_reward, 0)) as reward_total, count(distinct ic.holder_address) as address_total").
		Joins("left join pump_invite_reward as ii on ic.invite_address = ii.invite_address").
		Group("ic.invite_address")

	if params.InviteAddress != "" {
		subQuery = subQuery.Where("ic.invite_address = ?", params.InviteAddress)
	}

	err := subQuery.Find(&total).Error
	if err != nil {
		println(err.Error())
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = total

	c.JSON(http.StatusOK, result)
}
