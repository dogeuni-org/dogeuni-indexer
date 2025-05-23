package main

import (
	"context"
	"dogeuni-indexer/config"
	"dogeuni-indexer/explorer"
	"dogeuni-indexer/router"
	"dogeuni-indexer/router_v3"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/storage_v3"
	"github.com/dogecoinw/doged/rpcclient"
	"github.com/dogecoinw/go-dogecoin/log"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	cfg config.Config
)

func main() {

	// Load configuration file
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

	connCfg := &rpcclient.ConnConfig{
		Host:         cfg.Chain.Rpc,
		Endpoint:     "ws",
		User:         cfg.Chain.UserName,
		Pass:         cfg.Chain.PassWord,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	rpcClient, _ := rpcclient.New(connCfg, nil)

	ipfs := shell.NewShell(cfg.Ipfs)

	if cfg.Explorer.Switch {
		exp := explorer.NewExplorer(ctx, wg, rpcClient, dbClient, ipfs, cfg.Explorer.FromBlock)
		wg.Add(1)
		go exp.Start()
	}

	if cfg.HttpServer.Switch {

		levelClient := storage.NewLevelDB(cfg.LevelDB)

		grt := gin.Default()
		grt.Use(func(c *gin.Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
			c.Writer.Header().Set("Access-Control-Max-Age", "3600")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(200)
				return
			}
			c.Next()
		})

		rt := router_v3.NewRouter(mysqlClient, dbClient, levelClient, rpcClient, ipfs)

		grt.POST("/v3/info/lastnumber", rt.LastNumber)

		grt.POST("/v3/drc20/order", rt.FindOrdersIndex)

		grt.POST("/v3/drc20/all", rt.FindDrc20All)
		grt.POST("/v3/drc20/tick/address", rt.FindDrc20TickAddress)
		grt.POST("/v3/drc20/popular", rt.FindDrc20Popular)
		grt.POST("/v3/drc20/tick", rt.FindDrc20ByTick)
		grt.POST("/v3/drc20/holders", rt.FindDrc20Holders)
		grt.POST("/v3/drc20/address", rt.FindDrc20sByAddress)
		grt.POST("/v3/drc20/address/tick", rt.FindDrc20sByAddressTick)
		grt.POST("/v3/drc20/orders", rt.FindOrders)
		grt.POST("/v3/drc20/order/id", rt.FindOrdersByid)

		grt.POST("/v3/orders/address", rt.FindOrderByAddress)
		grt.POST("/v3/orders/hash", rt.FindOrdersHash)
		grt.POST("/v3/orders/address/tick", rt.FindOrdersByTick)

		grt.POST("/v3/address/ogall", rt.FindOgAddressAll)

		// swap
		grt.POST("/v3/swap/getreserves", rt.SwapGetReserves)
		grt.POST("/v3/swap/getreserves/all", rt.SwapGetReservesAll)
		grt.POST("/v3/swap/getliquidity", rt.SwapGetLiquidity)
		grt.POST("/v3/swap/order/id", rt.SwapInfoById)
		grt.POST("/v3/swap/order", rt.SwapInfo)
		grt.POST("/v3/swap/price", rt.SwapPrice)
		grt.POST("/v3/swap/k", rt.SwapSummaryK)
		grt.POST("/v3/swap/tvl", rt.SwapSummaryTvl)
		grt.POST("/v3/swap/k/new", rt.SwapSummaryKNew)
		grt.POST("/v3/swap/tvl/all", rt.SwapSummaryTvlAll)
		grt.POST("/v3/swap/summaryall", rt.SwapSummaryAll)
		grt.POST("/v3/swap/summary/tick", rt.SwapSummaryByTick)
		grt.POST("/v3/swap/pair/tick", rt.SwapPairByTick)
		grt.POST("/v3/swap/pair/all", rt.SwapPairAll)

		// exchange
		grt.POST("/v3/exchange/collect", rt.ExchangeCollect)
		grt.POST("/v3/exchange/order", rt.ExchangeInfo)
		grt.POST("/v3/exchange/order/tick", rt.ExchangeInfoByTick)
		grt.POST("/v3/exchange/summary", rt.ExchangeSummary)
		grt.POST("/v3/exchange/summaryall", rt.ExchangeSummaryAll)
		grt.POST("/v3/exchange/summary/tick", rt.ExchangeSummaryByTick)
		grt.POST("/v3/exchange/k", rt.ExchangeSummaryK)

		// box
		grt.POST("/v3/box/order", rt.BoxInfo)
		grt.POST("/v3/box/collect", rt.BoxCollect)

		// lp
		grt.POST("/v3/stake/all", rt.StakeAll)
		grt.POST("/v3/stake/tick", rt.StakeByTick)
		grt.POST("/v3/stake/reward", rt.StakeReward)
		grt.POST("/v3/stake/holders", rt.StakeHolders)
		grt.POST("/v3/stake/address/tick", rt.StakeByAddressTick)
		grt.POST("/v3/stake/order/id", rt.StakeInfoById)
		grt.POST("/v3/stake/order", rt.StakeInfo)

		// dogew
		grt.POST("/v3/dogew/order/id", rt.WDogeInfoById)
		grt.POST("/v3/dogew/order", rt.WDogeInfo)

		// nft
		grt.POST("/v3/nft/all", rt.FindNftAll)
		grt.POST("/v3/nft/tick", rt.FindNftByTick)
		grt.POST("/v3/nft/tick/id", rt.FindNftByTickAndId)
		grt.POST("/v3/nft/holders", rt.FindNftHolders)
		grt.POST("/v3/nft/address/tick", rt.FindNftByAddress)
		grt.POST("/v3/nft/order/id", rt.NftInfoById)
		grt.POST("/v3/nft/order", rt.NftInfo)

		grt.POST("/v3/tx/broadcast", rt.TxBroadcast)
		// v4
		v4 := grt.Group("/v4")
		{

			infoRouter := router.NewInfoRouter(dbClient, rpcClient, levelClient, ipfs)
			v4.POST("/info/lastnumber", infoRouter.LastNumber)
			v4.POST("/info/blocknumber", infoRouter.BlockNumber)

			drc20Router := router.NewDrc20Router(dbClient, rpcClient, levelClient, ipfs)
			v4.POST("/drc20/order", drc20Router.Order)
			v4.POST("/drc20/collect", drc20Router.Collect)
			v4.POST("/drc20/collect-address", drc20Router.CollectAddress)
			v4.POST("/drc20/history", drc20Router.History)

			swapRouter := router.NewSwapRouter(dbClient, rpcClient)
			v4.POST("/swap/order", swapRouter.Order)
			v4.POST("/swap/liquidity", swapRouter.SwapLiquidity)
			v4.POST("/swap/liquidity/address", swapRouter.SwapLiquidityHolder)
			v4.POST("/swap/price", swapRouter.SwapPrice)
			v4.POST("/swap/k", swapRouter.SwapK)
			v4.POST("/swap/tvl", swapRouter.SwapTvl)
			v4.POST("/swap/tvl/total", swapRouter.SwapSummaryTvlTotal)
			v4.POST("/swap/summary", swapRouter.SwapSummary)
			v4.POST("/swap/pair", swapRouter.SwapPair)

			// exchange
			exchangeRouter := router.NewExchangeRouter(dbClient, rpcClient)
			v4.POST("/exchange/order", exchangeRouter.Order)
			v4.POST("/exchange/collect", exchangeRouter.Collect)
			v4.POST("/exchange/summary", exchangeRouter.Summary)
			v4.POST("/exchange/summary/total", exchangeRouter.SummaryTotal)
			v4.POST("/exchange/k", exchangeRouter.SummaryK)

			// box
			boxRouter := router.NewBoxRouter(dbClient, rpcClient)
			v4.POST("/box/order", boxRouter.Order)
			v4.POST("/box/collect", boxRouter.Collect)

			// wdoge
			wdogeRouter := router.NewWdogeRouter(dbClient, rpcClient)
			v4.POST("/wdoge/order", wdogeRouter.Order)

			// stake
			stakeRouter := router.NewStakeRouter(dbClient, rpcClient)
			v4.POST("/stake/order", stakeRouter.Order)
			v4.POST("/stake/collect", stakeRouter.Collect)
			v4.POST("/stake/collect-address", stakeRouter.CollectAddress)
			v4.POST("/stake/reward", stakeRouter.Reward)
			v4.POST("/stake/total", stakeRouter.Total)

			// nft
			nftRouter := router.NewNftRouter(dbClient, rpcClient)
			v4.POST("/nft/order", nftRouter.Order)
			v4.POST("/nft/collect", nftRouter.Collect)
			v4.POST("/nft/collect-address", nftRouter.CollectAddress)

			// file
			fileRouter := router.NewFileRouter(dbClient, rpcClient, ipfs)
			v4.POST("/file/order", fileRouter.Order)
			v4.POST("/file/collect-address", fileRouter.CollectAddress)

			v4.POST("/file/upload/meta", fileRouter.UploadMeta)
			v4.POST("/file/upload/inscriptions/meta", fileRouter.UploadInscriptionsMeta)

			v4.POST("/file/collections", fileRouter.Collections)
			v4.POST("/file/collections/inscriptions", fileRouter.CollectionsInscriptions)
			v4.POST("/file/collections/attributes", fileRouter.CollectionsAttributes)

			// file exchange
			fileExchangeRouter := router.NewFileExchangeRouter(dbClient, rpcClient, ipfs)
			v4.POST("/file-exchange/order", fileExchangeRouter.Order)
			v4.POST("/file-exchange/activity", fileExchangeRouter.Activity)
			v4.POST("/file-exchange/collect", fileExchangeRouter.Collect)
			v4.POST("/file-exchange/summary/all", fileExchangeRouter.SummaryAll)
			v4.POST("/file-exchange/summary/nft/all", fileExchangeRouter.SummaryAll)
			v4.POST("/file-exchange/inscriptions", fileExchangeRouter.Inscriptions)

			// cross
			crossRouter := router.NewCrossRouter(dbClient, rpcClient)
			v4.POST("/cross/order", crossRouter.Order)
			v4.POST("/cross/collect", crossRouter.Collect)
			// meme20
			meme20Router := router.NewMeme20Router(dbClient, rpcClient, levelClient)
			v4.POST("/meme20/order", meme20Router.Order)
			v4.POST("/meme20/collect", meme20Router.Collect)
			v4.POST("/meme20/collect-address", meme20Router.CollectAddress)
			v4.POST("/meme20/history", meme20Router.History)

			// pump
			pumpRouter := router.NewPumpRouter(dbClient, rpcClient)
			v4.POST("/pump/order", pumpRouter.Order)
			v4.POST("/pump/mergeorder", pumpRouter.MergeOrder)
			v4.POST("/pump/liquidity", pumpRouter.Liquidity)
			v4.POST("/pump/board", pumpRouter.Board)
			v4.POST("/pump/k", pumpRouter.K)
			v4.POST("/pump/king", pumpRouter.King)

			// swapv2
			swapV2Router := router.NewSwapV2Router(dbClient, rpcClient)
			v4.POST("/swap_v2/order", swapV2Router.Order)
			v4.POST("/swap_v2/liquidity", swapV2Router.Liquidity)
			v4.POST("/swap_v2/liquidity/address", swapV2Router.SwapLiquidityHolder)
			v4.POST("/swap_v2/price", swapV2Router.SwapPrice)
			v4.POST("/swap_v2/k", pumpRouter.K)

			// invite
			inviteRouter := router.NewInviteRouter(dbClient, rpcClient, levelClient)
			v4.POST("/invite/order", inviteRouter.Order)
			v4.POST("/invite/collect", inviteRouter.Collect)
			v4.POST("/invite/pump-reword", inviteRouter.PumpReward)
			v4.POST("/invite/pump-reword-total", inviteRouter.PumpRewardTotal)
		}

		err := grt.Run(cfg.HttpServer.Server)
		if err != nil {
			panic(err)
		}
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		println("\nReceived an interrupt, stopping services...")
		cancel()
	}()
	wg.Wait()
}
