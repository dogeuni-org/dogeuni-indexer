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

// ABI by contract id (if developer has shared it)
// Note: ABI is a compile-time artifact, not required for contract functionality
func (r *CardityRouter) ABI(c *gin.Context) {
	id := c.Param("id")
	m := &models.CardityContract{}
	if err := r.dbc.DB.Where("contract_id = ?", id).First(m).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "contract not found", "data": nil, "total": 0})
		return
	}
	
	if m.AbiJSON == "" {
		c.JSON(http.StatusOK, gin.H{
			"code": 404, 
			"msg": "ABI not available for this contract", 
			"data": nil, 
			"total": 0,
			"note": "ABI is a compile-time artifact that developers manage locally. Index service only handles on-chain data.",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200, 
		"msg": "success", 
		"data": gin.H{
			"abi_json": m.AbiJSON, 
			"abi_hash": m.AbiHash,
			"note": "ABI found in database (if previously stored)",
		}, 
		"total": 1,
	})
}

// Contracts list with enhanced filters
func (r *CardityRouter) Contracts(c *gin.Context) {
	type req struct {
		Creator     string `json:"creator"`
		Protocol    string `json:"protocol"`
		Version     string `json:"version"`
		CarcSHA256  string `json:"carc_sha256"`
		AbiHash     string `json:"abi_hash"`
		ContractRef string `json:"contract_ref"`
		Offset      int    `json:"offset"`
		Limit       int    `json:"limit"`
		SinceBlock  int64  `json:"since_block"`
		UntilBlock  int64  `json:"until_block"`
	}
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}

	q := r.dbc.DB.Model(&models.CardityContract{})
	if p.Creator != "" {
		q = q.Where("creator = ?", p.Creator)
	}
	if p.Protocol != "" {
		q = q.Where("protocol = ?", p.Protocol)
	}
	if p.Version != "" {
		q = q.Where("version = ?", p.Version)
	}
	if p.CarcSHA256 != "" {
		q = q.Where("carc_sha256 = ?", p.CarcSHA256)
	}
	if p.AbiHash != "" {
		q = q.Where("abi_hash = ?", p.AbiHash)
	}
	if p.ContractRef != "" {
		q = q.Where("contract_ref = ?", p.ContractRef)
	}
	if p.SinceBlock > 0 {
		q = q.Where("block_number >= ?", p.SinceBlock)
	}
	if p.UntilBlock > 0 {
		q = q.Where("block_number <= ?", p.UntilBlock)
	}
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
		ContractId  string `json:"contract_id"`
		Method      string `json:"method"`
		MethodFQN   string `json:"method_fqn"`
		FromAddress string `json:"from_address"`
		Offset      int    `json:"offset"`
		Limit       int    `json:"limit"`
		SinceBlock  int64  `json:"since_block"`
		UntilBlock  int64  `json:"until_block"`
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
	if p.MethodFQN != "" {
		q = q.Where("method_fqn = ?", p.MethodFQN)
	}
	if p.FromAddress != "" {
		q = q.Where("from_address = ?", p.FromAddress)
	}
	if p.SinceBlock > 0 {
		q = q.Where("block_number >= ?", p.SinceBlock)
	}
	if p.UntilBlock > 0 {
		q = q.Where("block_number <= ?", p.UntilBlock)
	}
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
	cursorStr := c.Query("cursor_id")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := r.dbc.DB.Model(&models.CardityInvocationLog{}).Where("contract_id = ?", contractId)
	if methodFQN != "" {
		q = q.Where("method_fqn = ?", methodFQN)
	}
	if fromAddress != "" {
		q = q.Where("from_address = ?", fromAddress)
	}
	if sinceStr != "" {
		if v, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			q = q.Where("block_number >= ?", v)
		}
	}
	if untilStr != "" {
		if v, err := strconv.ParseInt(untilStr, 10, 64); err == nil {
			q = q.Where("block_number <= ?", v)
		}
	}
	if cursorStr != "" {
		if v, err := strconv.ParseInt(cursorStr, 10, 64); err == nil {
			q = q.Where("id < ?", v)
		}
	}
	var total int64
	_ = q.Count(&total).Error
	list := make([]*models.CardityInvocationLog, 0)
	_ = q.Order("id desc").Offset(offset).Limit(limit).Find(&list).Error
	nextCursor := ""
	if len(list) > 0 {
		nextCursor = strconv.FormatInt(list[len(list)-1].Id, 10)
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": list, "total": total, "next_cursor": nextCursor})
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
	if p.SinceBlock > 0 {
		q = q.Where("block_number >= ?", p.SinceBlock)
	}
	if p.UntilBlock > 0 {
		q = q.Where("block_number <= ?", p.UntilBlock)
	}
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
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}
	q := r.dbc.DB.Model(&models.CardityPackage{})
	if p.PackageId != "" {
		q = q.Where("package_id = ?", p.PackageId)
	}
	if p.SinceBlock > 0 {
		q = q.Where("block_number >= ?", p.SinceBlock)
	}
	if p.UntilBlock > 0 {
		q = q.Where("block_number <= ?", p.UntilBlock)
	}
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
	if p.SinceBlock > 0 {
		q = q.Where("block_number >= ?", p.SinceBlock)
	}
	if p.UntilBlock > 0 {
		q = q.Where("block_number <= ?", p.UntilBlock)
	}
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

// SearchByABI allows searching contracts by ABI content for better discovery
// Note: ABI is used to query index content, not uploaded to index
func (r *CardityRouter) SearchByABI(c *gin.Context) {
	type req struct {
		AbiHash     string `json:"abi_hash"`
		MethodName  string `json:"method_name"` // Search contracts with specific method
		EventName   string `json:"event_name"`  // Search contracts with specific event
		Limit       int    `json:"limit"`
		Offset      int    `json:"offset"`
	}
	
	var p req
	_ = c.ShouldBindJSON(&p)
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	
	q := r.dbc.DB.Model(&models.CardityContract{})
	
	if p.AbiHash != "" {
		q = q.Where("abi_hash = ?", p.AbiHash)
	}
	
	if p.MethodName != "" {
		// Search in ABI JSON for method names (only if ABI exists)
		q = q.Where("abi_json LIKE ? AND abi_json != ''", "%"+p.MethodName+"%")
	}
	
	if p.EventName != "" {
		// Search in ABI JSON for event names (only if ABI exists)
		q = q.Where("abi_json LIKE ? AND abi_json != ''", "%"+p.EventName+"%")
	}
	
	var total int64
	_ = q.Count(&total).Error
	
	results := make([]*models.CardityContract, 0)
	_ = q.Offset(p.Offset).Limit(p.Limit).Order("block_number desc, id desc").Find(&results).Error
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200, 
		"msg": "success", 
		"data": results, 
		"total": total,
		"filters": gin.H{
			"abi_hash":    p.AbiHash,
			"method_name": p.MethodName,
			"event_name":  p.EventName,
		},
		"note": "ABI search helps discover contracts by their interface. ABI is used to query index content.",
	})
}

// GetABIStats returns statistics about ABI coverage for discovery purposes
// Note: ABI helps users discover and interact with contracts
func (r *CardityRouter) GetABIStats(c *gin.Context) {
	var stats struct {
		TotalContracts int64 `json:"total_contracts"`
		WithABI        int64 `json:"with_abi"`
		WithoutABI     int64 `json:"without_abi"`
		AbiCoverage    float64 `json:"abi_coverage"`
		BySourceType   map[string]int64 `json:"by_source_type"`
		Note           string `json:"note"`
	}
	
	// Total contracts
	r.dbc.DB.Model(&models.CardityContract{}).Count(&stats.TotalContracts)
	
	// Contracts with ABI (for discovery)
	r.dbc.DB.Model(&models.CardityContract{}).Where("abi_json != '' AND abi_json IS NOT NULL").Count(&stats.WithABI)
	
	// Contracts without ABI (normal case)
	stats.WithoutABI = stats.TotalContracts - stats.WithABI
	
	// Calculate coverage percentage
	if stats.TotalContracts > 0 {
		stats.AbiCoverage = float64(stats.WithABI) / float64(stats.TotalContracts) * 100
	}
	
	// By source type
	stats.BySourceType = make(map[string]int64)
	var sourceTypes []struct {
		SourceType string `json:"source_type"`
		Count      int64  `json:"count"`
	}
	r.dbc.DB.Model(&models.CardityContract{}).
		Select("abi_source_type, count(*) as count").
		Where("abi_source_type != '' AND abi_source_type IS NOT NULL").
		Group("abi_source_type").
		Find(&sourceTypes)
	
	for _, st := range sourceTypes {
		stats.BySourceType[st.SourceType] = st.Count
	}
	
	stats.Note = "ABI helps users discover and interact with contracts. Coverage shows contracts with discoverable interfaces."
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200, 
		"msg": "success", 
		"data": stats,
	})
}
