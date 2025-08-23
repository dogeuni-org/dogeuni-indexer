package router_v3

import (
	"bytes"
	"dogeuni-indexer/utils"
	"encoding/hex"
	"fmt"
	"github.com/dogecoinw/doged/wire"
)

// TxBroadcastWasm is a WASM-compatible version of TxBroadcast
func (r *Router) TxBroadcastWasm(txHex string) (*utils.HttpResult, error) {
	bytesData, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, err
	}

	msgTx := new(wire.MsgTx)
	err = msgTx.Deserialize(bytes.NewReader(bytesData))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	txhash, err := r.node.SendRawTransaction(msgTx, true)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	data["tx_hash"] = txhash.String()
	result := &utils.HttpResult{
		Code: 200,
		Msg:  "success",
		Data: data,
	}
	
	return result, nil
}