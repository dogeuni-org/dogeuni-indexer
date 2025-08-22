package router

import (
	"bytes"
	"context"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"encoding/hex"
	"fmt"
	"github.com/dogecoinw/doged/rpcclient"
	"github.com/dogecoinw/doged/wire"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// simple token bucket
var cardityTokens = make(chan struct{}, 50)

func init() {
	for i := 0; i < cap(cardityTokens); i++ { cardityTokens <- struct{}{} }
}

func CardityRateLimitTimeout() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/v4/cardity/contracts" ||
			c.Request.URL.Path == "/v4/cardity/invocations" ||
			c.Request.URL.Path == "/v4/cardity/events" ||
			c.Request.URL.Path == "/v4/cardity/packages" ||
			c.Request.URL.Path == "/v4/cardity/modules" ||
			len(c.Request.URL.Path) >= len("/v4/cardity/") && c.Request.URL.Path[:len("/v4/cardity/")] == "/v4/cardity/" {
			select {
			case <-cardityTokens:
				defer func() { cardityTokens <- struct{}{} }()
				ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
				defer cancel()
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			case <-time.After(2 * time.Second):
				c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "msg": "rate limited", "data": nil})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

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
