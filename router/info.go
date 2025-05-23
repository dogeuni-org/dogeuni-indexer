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

type InfoRouter struct {
	dbc   *storage.DBClient
	node  *rpcclient.Client
	ipfs  *shell.Shell
	level *storage.LevelDB
}

func NewInfoRouter(db *storage.DBClient, node *rpcclient.Client, level *storage.LevelDB, ipfs *shell.Shell) *InfoRouter {
	return &InfoRouter{
		dbc:   db,
		node:  node,
		ipfs:  ipfs,
		level: level,
	}
}

func (r *InfoRouter) LastNumber(c *gin.Context) {
	maxHeight := 0
	err := r.dbc.DB.Model(&models.Block{}).Select("max(block_number)").Scan(&maxHeight).Error
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusOK, result)
		return
	}

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = maxHeight
	c.JSON(http.StatusOK, result)
}

func (r *InfoRouter) BlockNumber(c *gin.Context) {

	maxHeight := 0
	err := r.dbc.DB.Model(&models.Block{}).Select("max(block_number)").Scan(&maxHeight).Error
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusOK, result)
		return
	}

	chainHeight, err := r.node.GetBlockCount()
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = "server error"
		c.JSON(http.StatusOK, result)
		return
	}

	data := make(map[string]interface{})
	data["index_height"] = maxHeight
	data["chain_height"] = chainHeight

	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = data
	c.JSON(http.StatusOK, result)
}
