package main

import (
	"context"
	"dogeuni-indexer/config"
	"dogeuni-indexer/explorer"
	"dogeuni-indexer/metrics"
	"dogeuni-indexer/router"
	"dogeuni-indexer/router_v3"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/storage_v3"
	"github.com/dogecoinw/doged/rpcclient"
	"github.com/dogecoinw/go-dogecoin/log"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	shell "github.com/ipfs/go-ipfs-api"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	cfg config.Config
)

func main() {

	config.LoadConfig(&cfg, "")

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	glogger.Verbosity(log.Lvl(cfg.DebugLevel))
	log.Root().SetHandler(glogger)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	mysqlClient := storage_v3.NewSqliteClient(cfg.Sqlite)

	var dbClient *storage.DBClient
	if cfg.Sqlite.Switch {
		dbClient = storage.NewSqliteClient(cfg.Sqlite)
	} else {
		dbClient = storage.NewMysqlClient(cfg.Mysql)
	}

	_ = dbClient.AutoMigrateCardity()

	connCfg := &rpcclient.ConnConfig{
		Host:         cfg.Chain.Rpc,
		Endpoint:     "ws",
		User:         cfg.Chain.UserName,
		Pass:         cfg.Chain.PassWord,
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	rpcClient, _ := rpcclient.New(connCfg, nil)

	ipfs := shell.NewShell(cfg.Ipfs)

	if cfg.Explorer.Switch {
		exp := explorer.NewExplorer(ctx, wg, rpcClient, dbClient, ipfs, cfg.Explorer.FromBlock)
		wg.Add(1)
		go exp.Start()
	}

	if cfg.HttpServer.Switch {
		metrics.MustRegister()

		levelClient := storage.NewLevelDB(cfg.LevelDB)
		grt := gin.Default()
		grt.Use(func(c *gin.Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
			c.Writer.Header().Set("Access-Control-Max-Age", "3600")
			if c.Request.Method == "OPTIONS" { c.AbortWithStatus(200); return }
			c.Next()
		})

		grt.GET("/metrics", func(c *gin.Context) {
			promhttp.Handler().ServeHTTP(c.Writer, c.Request)
		})

		rt := router_v3.NewRouter(mysqlClient, dbClient, levelClient, rpcClient, ipfs)
		grt.POST("/v3/info/lastnumber", rt.LastNumber)

		v4 := grt.Group("/v4")
		{
			cardityRouter := router.NewCardityRouter(dbClient)
			v4.POST("/cardity/contracts", cardityRouter.Contracts)
			v4.POST("/cardity/invocations", cardityRouter.Invocations)
			v4.POST("/cardity/events", cardityRouter.Events)
			v4.POST("/cardity/packages", cardityRouter.Packages)
			v4.POST("/cardity/modules", cardityRouter.Modules)
			v4.GET("/cardity/abi/:id", cardityRouter.ABI)
			v4.GET("/cardity/contract/:id", cardityRouter.Contract)
			v4.GET("/cardity/invocations/:contractId", cardityRouter.InvocationsByContract)
			v4.GET("/cardity/modules/:packageId", cardityRouter.ModulesByPackage)
		}

		if err := grt.Run(cfg.HttpServer.Server); err != nil { panic(err) }
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() { <-c; println("\nReceived an interrupt, stopping services..."); cancel() }()
	wg.Wait()
}
