package router

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CardityRouter struct {
	dbc *storage.DBClient
}

func NewCardityRouter(db *storage.DBClient) *CardityRouter {
	return &CardityRouter{dbc: db}
}

// Contracts list
func (r *CardityRouter) Contracts(c *gin.Context) {
	type req struct {
		Protocol string `json:"protocol"`
		Offset   int    `json:"offset"`
		Limit    int    `json:"limit"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}

	q := r.dbc.DB.Model(&models.CardityContract{})
	if p.Protocol != "" {
		q = q.Where("protocol = ?", p.Protocol)
	}
	var total int64
	_ = q.Count(&total).Error
	results := make([]*models.CardityContract, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("id desc").Find(&results).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": results, "total": total})
}

// Invocations list
func (r *CardityRouter) Invocations(c *gin.Context) {
	type req struct {
		ContractId string `json:"contract_id"`
		Method     string `json:"method"`
		Offset     int    `json:"offset"`
		Limit      int    `json:"limit"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}

	q := r.dbc.DB.Model(&models.CardityInvocationLog{})
	if p.ContractId != "" {
		q = q.Where("contract_id = ?", p.ContractId)
	}
	if p.Method != "" {
		q = q.Where("method = ?", p.Method)
	}
	var total int64
	_ = q.Count(&total).Error
	results := make([]*models.CardityInvocationLog, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("id desc").Find(&results).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": results, "total": total})
}

// Events list
func (r *CardityRouter) Events(c *gin.Context) {
	type req struct {
		ContractId string `json:"contract_id"`
		EventName  string `json:"event_name"`
		Offset     int    `json:"offset"`
		Limit      int    `json:"limit"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}

	q := r.dbc.DB.Model(&models.CardityEventLog{})
	if p.ContractId != "" {
		q = q.Where("contract_id = ?", p.ContractId)
	}
	if p.EventName != "" {
		q = q.Where("event_name = ?", p.EventName)
	}
	var total int64
	_ = q.Count(&total).Error
	results := make([]*models.CardityEventLog, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("id desc").Find(&results).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": results, "total": total})
}

// Packages list
func (r *CardityRouter) Packages(c *gin.Context) {
	type req struct {
		PackageId string `json:"package_id"`
		Offset    int    `json:"offset"`
		Limit     int    `json:"limit"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}
	q := r.dbc.DB.Model(&models.CardityPackage{})
	if p.PackageId != "" {
		q = q.Where("package_id = ?", p.PackageId)
	}
	var total int64
	_ = q.Count(&total).Error
	list := make([]*models.CardityPackage, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("id desc").Find(&list).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": list, "total": total})
}

// Modules list
func (r *CardityRouter) Modules(c *gin.Context) {
	type req struct {
		PackageId string `json:"package_id"`
		Name      string `json:"name"`
		Offset    int    `json:"offset"`
		Limit     int    `json:"limit"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}
	q := r.dbc.DB.Model(&models.CardityModule{})
	if p.PackageId != "" {
		q = q.Where("package_id = ?", p.PackageId)
	}
	if p.Name != "" {
		q = q.Where("name = ?", p.Name)
	}
	var total int64
	_ = q.Count(&total).Error
	list := make([]*models.CardityModule, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("id desc").Find(&list).Error
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": list, "total": total})
}
