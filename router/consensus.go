package router

import (
	"math/big"
	"net/http"

	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"

	"github.com/dogecoinw/doged/rpcclient"
	"github.com/gin-gonic/gin"
)

type ConsensusRouter struct {
	dbc  *storage.DBClient
	node *rpcclient.Client
}

func NewConsensusRouter(dbc *storage.DBClient, node *rpcclient.Client) *ConsensusRouter {
	return &ConsensusRouter{dbc: dbc, node: node}
}

// Order 查询共识订单（consensus_info）
func (r *ConsensusRouter) Order(c *gin.Context) {
	type params struct {
		OrderId       string `json:"order_id"`
		Op            string `json:"op"`
		StakeId       string `json:"stake_id"`
		HolderAddress string `json:"holder_address"`
		BlockNumber   int64  `json:"block_number"`
		Limit         int    `json:"limit"`
		OffSet        int    `json:"offset"`
	}

	p := &params{Limit: 10, OffSet: 0}
	if err := c.ShouldBindJSON(p); err != nil {
		result := &utils.HttpResult{Code: 500, Msg: err.Error()}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	filter := &models.ConsensusInfo{
		OrderId:       p.OrderId,
		Op:            p.Op,
		StakeId:       p.StakeId,
		HolderAddress: p.HolderAddress,
		BlockNumber:   p.BlockNumber,
	}

	infos := make([]*models.ConsensusInfo, 0)
	total := int64(0)
	err := r.dbc.DB.Model(&models.ConsensusInfo{}).Where(filter).Order("id desc").Count(&total).Limit(p.Limit).Offset(p.OffSet).Find(&infos).Error
	if err != nil {
		result := &utils.HttpResult{Code: 500, Msg: err.Error()}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	result := &utils.HttpResult{Code: 200, Msg: "success", Data: infos, Total: total}
	c.JSON(http.StatusOK, result)
}

// Records 查询独立质押记录（consensus_stake_record）
func (r *ConsensusRouter) Records(c *gin.Context) {
	type params struct {
		HolderAddress string `json:"holder_address"`
		Status        string `json:"status"` // active/closed，可选
		Limit         int    `json:"limit"`
		OffSet        int    `json:"offset"`
	}

	p := &params{Limit: 10, OffSet: 0}
	if err := c.ShouldBindJSON(p); err != nil {
		result := &utils.HttpResult{Code: 500, Msg: err.Error()}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	where := map[string]interface{}{}
	if p.HolderAddress != "" {
		where["holder_address"] = p.HolderAddress
	}
	if p.Status != "" {
		where["status"] = p.Status
	}

	records := make([]*models.ConsensusStakeRecord, 0)
	total := int64(0)
	err := r.dbc.DB.Model(&models.ConsensusStakeRecord{}).Where(where).Order("id desc").Count(&total).Limit(p.Limit).Offset(p.OffSet).Find(&records).Error
	if err != nil {
		result := &utils.HttpResult{Code: 500, Msg: err.Error()}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	result := &utils.HttpResult{Code: 200, Msg: "success", Data: records, Total: total}
	c.JSON(http.StatusOK, result)
}

// Score 计算单笔记录当前积分（含衰减）
func (r *ConsensusRouter) Score(c *gin.Context) {
	type params struct {
		StakeId      string  `json:"stake_id"`
		CurrentBlock int64   `json:"current_block"`
		BlocksPerDay int64   `json:"blocks_per_day"`
		Lambda       float64 `json:"lambda"`
		Beta         float64 `json:"beta"`
	}

	p := &params{BlocksPerDay: 1440, Lambda: 0.15, Beta: 0.2}
	if err := c.ShouldBindJSON(p); err != nil {
		result := &utils.HttpResult{Code: 500, Msg: err.Error()}
		c.JSON(http.StatusBadRequest, result)
		return
	}
	if p.StakeId == "" {
		result := &utils.HttpResult{Code: 500, Msg: "stake_id required"}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	// 若未传当前高度，则从节点获取最新高度
	if p.CurrentBlock == 0 {
		bc, err := r.node.GetBlockCount()
		if err != nil {
			result := &utils.HttpResult{Code: 500, Msg: err.Error()}
			c.JSON(http.StatusBadRequest, result)
			return
		}
		p.CurrentBlock = bc
	}

	rec := &models.ConsensusStakeRecord{}
	if err := r.dbc.DB.Where("stake_id = ?", p.StakeId).First(rec).Error; err != nil {
		result := &utils.HttpResult{Code: 500, Msg: err.Error()}
		c.JSON(http.StatusBadRequest, result)
		return
	}

	var score *big.Int
	if rec.Status == "active" {
		score = r.dbc.CalculateConsensusScore(rec.Amt, rec.StakeBlock, p.CurrentBlock)
	} else {
		score = r.dbc.GetConsensusRecordDecayedScore(rec, p.CurrentBlock, p.BlocksPerDay, p.Lambda, p.Beta)
	}

	// 返回整数积分
	result := &utils.HttpResult{Code: 200, Msg: "success", Data: map[string]interface{}{
		"stake_id": p.StakeId,
		"score":    score.String(),
		"status":   rec.Status,
	}}
	c.JSON(http.StatusOK, result)
}
