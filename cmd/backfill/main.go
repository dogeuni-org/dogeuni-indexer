package main

import (
	"dogeuni-indexer/config"
	"dogeuni-indexer/storage"
	bf "dogeuni-indexer/tools/backfill"
	"flag"
	"fmt"
)

func main() {
	var (
		cfgFile   string
		batchSize int
		target    string
	)
	flag.StringVar(&cfgFile, "config", "config.json", "config file path")
	flag.IntVar(&batchSize, "batch", 1000, "batch size")
	flag.StringVar(&target, "target", "all", "all|drc20|meme20")
	flag.Parse()

	var cfg config.Config
	config.LoadConfig(&cfg, cfgFile)

	var db *storage.DBClient
	if cfg.Sqlite.Switch {
		db = storage.NewSqliteClient(cfg.Sqlite)
	} else {
		db = storage.NewMysqlClient(cfg.Mysql)
	}
	_ = db.AutoMigrateCardity()

	var err error
	switch target {
	case "drc20":
		err = bf.BackfillDRC20(db, batchSize)
	case "meme20":
		err = bf.BackfillMeme20(db, batchSize)
	default:
		err = bf.BackfillAll(db, batchSize)
	}
	if err != nil {
		fmt.Println("backfill error:", err)
	} else {
		fmt.Println("backfill done")
	}
}
