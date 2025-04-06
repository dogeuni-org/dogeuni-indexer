package router

import (
	"dogeuni-indexer/storage"
	"dogeuni-indexer/verifys"
	"github.com/dogecoinw/doged/rpcclient"
)

type Router struct {
	dbc  *storage.DBClient
	node *rpcclient.Client

	verify *verifys.Verifys
}
