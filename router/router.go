package router

import (
	"bytes"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"encoding/hex"
	"fmt"
	"github.com/dogecoinw/doged/rpcclient"
	"github.com/dogecoinw/doged/wire"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Router struct {
	dbc  *storage.DBClient
	node *rpcclient.Client
}

func NewRouter(db *storage.DBClient, node *rpcclient.Client) *Router {
	return &Router{
		dbc:  db,
		node: node,
	}
}

func (r *Router) TxBroadcast(c *gin.Context) {
	type params struct {
		TxHex string `json:"tx_hex"`
	}

	p := &params{}
	if err := c.ShouldBindJSON(&p); err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusOK, result)
		return
	}

	bytesData, err := hex.DecodeString(p.TxHex)
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusOK, result)
		return
	}

	msgTx := new(wire.MsgTx)
	err = msgTx.Deserialize(bytes.NewReader(bytesData))
	if err != nil {
		fmt.Println(err)
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusOK, result)
		return
	}

	txhash, err := r.node.SendRawTransaction(msgTx, true)
	if err != nil {
		result := &utils.HttpResult{}
		result.Code = 500
		result.Msg = err.Error()
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	data := make(map[string]interface{})
	data["tx_hash"] = txhash.String()
	result := &utils.HttpResult{}
	result.Code = 200
	result.Msg = "success"
	result.Data = data
	c.JSON(http.StatusOK, result)
}
