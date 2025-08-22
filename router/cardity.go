package router

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CardityRouter struct {
	dbc *storage.DBClient
}

func NewCardityRouter(db *storage.DBClient) *CardityRouter {
	return &CardityRouter{dbc: db}
}

// ABI by contract id
func (r *CardityRouter) ABI(c *gin.Context) {
	id := c.Param("id")
	m := &models.CardityContract{}
	if err := r.dbc.DB.Where("contract_id = ?", id).First(m).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "not found", "data": nil, "total": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": gin.H{"abi_json": m.AbiJSON, "abi_hash": m.AbiHash}, "total": 1})
}

// Contracts list with enhanced filters
func (r *CardityRouter) Contracts(c *gin.Context) {
	type req struct {
		Creator    string `json:"creator"`
		Protocol   string `json:"protocol"`
		Version    string `json:"version"`
		CarcSHA256 string `json:"carc_sha256"`
		AbiHash    string `json:"abi_hash"`
		ContractRef string `json:"contract_ref"`
		Offset     int    `json:"offset"`
		Limit      int    `json:"limit"`
		SinceBlock int64  `json:"since_block"`
		UntilBlock int64  `json:"until_block"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 { p.Limit = 50 }

	q := r.dbc.DB.Model(&models.CardityContract{})
	if p.Creator != "" { q = q.Where("creator = ?", p.Creator) }
	if p.Protocol != "" { q = q.Where("protocol = ?", p.Protocol) }
	if p.Version != "" { q = q.Where("version = ?", p.Version) }
	if p.CarcSHA256 != "" { q = q.Where("carc_sha256 = ?", p.CarcSHA256) }
	if p.AbiHash != "" { q = q.Where("abi_hash = ?", p.AbiHash) }
	if p.ContractRef != "" { q = q.Where("contract_ref = ?", p.ContractRef) }
	if p.SinceBlock > 0 { q = q.Where("block_number >= ?", p.SinceBlock) }
	if p.UntilBlock > 0 { q = q.Where("block_number <= ?", p.UntilBlock) }
	var total int64
	_ = q.Count(&total).Error
	results := make([]*models.CardityContract, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("block_number desc, id desc").Find(&results).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": results, "total": total})
}

// GET /cardity/contract/:id
func (r *CardityRouter) Contract(c *gin.Context) {
	id := c.Param("id")
	m := &models.CardityContract{}
	if err := r.dbc.DB.Where("contract_id = ?", id).First(m).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "not found", "data": nil, "total": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": m, "total": 1})
}

// Invocations list with enhanced filters
func (r *CardityRouter) Invocations(c *gin.Context) {
	type req struct {
		ContractId string `json:"contract_id"`
		Method     string `json:"method"`
		MethodFQN  string `json:"method_fqn"`
		FromAddress string `json:"from_address"`
		Offset     int    `json:"offset"`
		Limit      int    `json:"limit"`
		SinceBlock int64  `json:"since_block"`
		UntilBlock int64  `json:"until_block"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 { p.Limit = 50 }

	q := r.dbc.DB.Model(&models.CardityInvocationLog{})
	if p.ContractId != "" { q = q.Where("contract_id = ?", p.ContractId) }
	if p.MethodFQN != "" { q = q.Where("method_fqn = ?", p.MethodFQN) } else if p.Method != "" { q = q.Where("method = ?", p.Method) }
	if p.FromAddress != "" { q = q.Where("from_address = ?", p.FromAddress) }
	if p.SinceBlock > 0 { q = q.Where("block_number >= ?", p.SinceBlock) }
	if p.UntilBlock > 0 { q = q.Where("block_number <= ?", p.UntilBlock) }
	var total int64
	_ = q.Count(&total).Error
	results := make([]*models.CardityInvocationLog, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("block_number desc, id desc").Find(&results).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": results, "total": total})
}

// GET /cardity/invocations/:contractId
func (r *CardityRouter) InvocationsByContract(c *gin.Context) {
	contractId := c.Param("contractId")
	methodFQN := c.Query("method_fqn")
	fromAddress := c.Query("from_address")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")
	sinceStr := c.Query("since_block")
	untilStr := c.Query("until_block")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 200 { limit = 50 }
	q := r.dbc.DB.Model(&models.CardityInvocationLog{}).Where("contract_id = ?", contractId)
	if methodFQN != "" { q = q.Where("method_fqn = ?", methodFQN) }
	if fromAddress != "" { q = q.Where("from_address = ?", fromAddress) }
	if sinceStr != "" { if v, err := strconv.ParseInt(sinceStr, 10, 64); err == nil { q = q.Where("block_number >= ?", v) } }
	if untilStr != "" { if v, err := strconv.ParseInt(untilStr, 10, 64); err == nil { q = q.Where("block_number <= ?", v) } }
	var total int64
	_ = q.Count(&total).Error
	list := make([]*models.CardityInvocationLog, 0)
	_ = q.Offset(offset).Limit(limit).Order("block_number desc, id desc").Find(&list).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": list, "total": total})
}

// Events list
func (r *CardityRouter) Events(c *gin.Context) {
	type req struct {
		ContractId string `json:"contract_id"`
		EventName  string `json:"event_name"`
		Offset     int    `json:"offset"`
		Limit      int    `json:"limit"`
		SinceBlock int64  `json:"since_block"`
		UntilBlock int64  `json:"until_block"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 { p.Limit = 50 }

	q := r.dbc.DB.Model(&models.CardityEventLog{})
	if p.ContractId != "" { q = q.Where("contract_id = ?", p.ContractId) }
	if p.EventName != "" { q = q.Where("event_name = ?", p.EventName) }
	if p.SinceBlock > 0 { q = q.Where("block_number >= ?", p.SinceBlock) }
	if p.UntilBlock > 0 { q = q.Where("block_number <= ?", p.UntilBlock) }
	var total int64
	_ = q.Count(&total).Error
	results := make([]*models.CardityEventLog, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("block_number desc, id desc").Find(&results).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": results, "total": total})
}

// Packages list (unchanged filters)
func (r *CardityRouter) Packages(c *gin.Context) {
	type req struct {
		PackageId  string `json:"package_id"`
		Offset     int    `json:"offset"`
		Limit      int    `json:"limit"`
		SinceBlock int64  `json:"since_block"`
		UntilBlock int64  `json:"until_block"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 { p.Limit = 50 }
	q := r.dbc.DB.Model(&models.CardityPackage{})
	if p.PackageId != "" { q = q.Where("package_id = ?", p.PackageId) }
	if p.SinceBlock > 0 { q = q.Where("block_number >= ?", p.SinceBlock) }
	if p.UntilBlock > 0 { q = q.Where("block_number <= ?", p.UntilBlock) }
	var total int64
	_ = q.Count(&total).Error
	list := make([]*models.CardityPackage, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("block_number desc, id desc").Find(&list).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": list, "total": total})
}

// Modules list (unchanged filters)
func (r *CardityRouter) Modules(c *gin.Context) {
	type req struct {
		PackageId  string `json:"package_id"`
		Name       string `json:"name"`
		Offset     int    `json:"offset"`
		Limit      int    `json:"limit"`
		SinceBlock int64  `json:"since_block"`
		UntilBlock int64  `json:"until_block"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 { p.Limit = 50 }
	q := r.dbc.DB.Model(&models.CardityModule{})
	if p.PackageId != "" { q = q.Where("package_id = ?", p.PackageId) }
	if p.Name != "" { q = q.Where("name = ?", p.Name) }
	if p.SinceBlock > 0 { q = q.Where("block_number >= ?", p.SinceBlock) }
	if p.UntilBlock > 0 { q = q.Where("block_number <= ?", p.UntilBlock) }
	var total int64
	_ = q.Count(&total).Error
	list := make([]*models.CardityModule, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("block_number desc, id desc").Find(&list).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": list, "total": total})
}

// GET /cardity/modules/:packageId
func (r *CardityRouter) ModulesByPackage(c *gin.Context) {
	packageId := c.Param("packageId")
	q := r.dbc.DB.Model(&models.CardityModule{}).Where("package_id = ?", packageId)
	var total int64
	_ = q.Count(&total).Error
	list := make([]*models.CardityModule, 0)
	_ = q.Order("block_number desc, id desc").Find(&list).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": list, "total": total})
}
