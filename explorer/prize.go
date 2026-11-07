package explorer

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/btcutil"
	"github.com/dogecoinw/doged/chaincfg"
	"github.com/dogecoinw/doged/chaincfg/chainhash"
	"github.com/dogecoinw/go-dogecoin/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// prize-v1, a generic prize pool. See docs/prize-v1.md.

func (e *Explorer) prizeDecode(tx *btcjson.TxRawResult, pushedData []byte, number int64) (*models.PrizeInfo, error) {

	if err := e.dropPending(&models.PrizeInfo{}, tx.Txid); err != nil {
		return nil, err
	}

	err := e.dbc.DB.Where("tx_hash = ?", tx.Txid).First(&models.PrizeInfo{}).Error
	if err == nil {
		return nil, fmt.Errorf("prize already exist %s", tx.Txid)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: dedup: %v", STORAGE_ERR, err)
	}

	param := &models.PrizeInscription{}
	if err := json.Unmarshal(pushedData, param); err != nil {
		return nil, fmt.Errorf("json.Unmarshal err: %s", err.Error())
	}

	prize, err := utils.ConvertPrize(param)
	if err != nil {
		return nil, fmt.Errorf("ConvertPrize err: %s", err.Error())
	}

	if len(tx.Vout) < 1 {
		return nil, fmt.Errorf("op error, vout length is not 0")
	}

	prize.OrderId = uuid.New().String()
	prize.FeeTxHash = tx.Vin[0].Txid
	prize.TxHash = tx.Txid
	prize.BlockHash = tx.BlockHash
	prize.BlockNumber = number
	prize.OrderStatus = 1

	if prize.Op == "create" {
		prize.PoolId = tx.Txid
	}

	prize.HolderAddress, err = outputAddress(tx, 0)
	if err != nil {
		return nil, err
	}

	txhash0, _ := chainhash.NewHashFromStr(tx.Vin[0].Txid)
	txRawResult0, err := e.node.GetRawTransactionVerboseBool(txhash0)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", CHAIN_NETWORK_ERR, err)
	}

	if addr, err := e.commitSender(txRawResult0, number); err != nil {
		return nil, err
	} else if prize.HolderAddress != addr {
		return nil, fmt.Errorf("The address is not the same as the previous transaction")
	}

	err = e.dbc.DB.Save(prize).Error
	if err != nil {
		return nil, fmt.Errorf("%w: save inscription: %v", STORAGE_ERR, err)
	}

	return prize, nil
}

// prizeAddress checks that address is a Dogecoin mainnet P2PKH or P2SH address in
// its canonical form.
func prizeAddress(address string) error {
	addr, err := btcutil.DecodeAddress(address, &chaincfg.MainNetParams)
	if err != nil {
		return fmt.Errorf("address %q: %v", address, err)
	}
	switch addr.(type) {
	case *btcutil.AddressPubKeyHash, *btcutil.AddressScriptHash:
	default:
		return fmt.Errorf("address %q is not P2PKH or P2SH", address)
	}
	if !addr.IsForNet(&chaincfg.MainNetParams) || addr.EncodeAddress() != address {
		return fmt.Errorf("address %q is not a canonical mainnet address", address)
	}
	return nil
}

// prizeEntries parses a settle's entries and checks the rules that need no state.
func prizeEntries(prize *models.PrizeInfo, reserve string) ([]models.PrizeEntry, error) {
	entries, err := utils.ParsePrizeEntries(prize.Entries)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 || len(entries) > storage.PrizeMaxEntries {
		return nil, fmt.Errorf("settle needs 1 to %d entries, got %d", storage.PrizeMaxEntries, len(entries))
	}
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if err := prizeAddress(entry.Address); err != nil {
			return nil, err
		}
		if entry.Address == reserve {
			return nil, fmt.Errorf("entry address %s is the pool reserve", entry.Address)
		}
		if seen[entry.Address] {
			return nil, fmt.Errorf("entry address %s is repeated", entry.Address)
		}
		seen[entry.Address] = true
	}
	return entries, nil
}

// executePrize applies one op. All its writes and order_status = 0 commit in one
// transaction; any failed rule rolls everything back.
func (e *Explorer) executePrize(prize *models.PrizeInfo) error {
	tx := e.dbc.DB.Begin()
	if err := e.prizeApply(tx, prize); err != nil {
		tx.Rollback()
		return fmt.Errorf("prize %s err: %s", prize.Op, err.Error())
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func (e *Explorer) prizeApply(tx *gorm.DB, prize *models.PrizeInfo) error {
	reserve, err := storage.PrizeReserve(prize.PoolId)
	if err != nil {
		return err
	}

	switch prize.Op {
	case "create":
		if err := prizeAddress(prize.Settler); err != nil {
			return fmt.Errorf("settler: %v", err)
		}
		err = e.dbc.PrizeCreate(tx, prize, reserve)
	case "deposit":
		err = e.dbc.PrizeDeposit(tx, prize)
	case "settle":
		entries, err := prizeEntries(prize, reserve)
		if err != nil {
			return err
		}
		err = e.dbc.PrizeSettle(tx, prize, entries)
		if err != nil {
			return err
		}
	case "claim":
		paid, err := e.dbc.PrizeClaim(tx, prize)
		if err != nil {
			return err
		}
		prize.Amt = (*models.Number)(paid)
	case "refund":
		paid, err := e.dbc.PrizeRefund(tx, prize)
		if err != nil {
			return err
		}
		prize.Amt = (*models.Number)(paid)
	case "withdraw":
		paid, err := e.dbc.PrizeWithdraw(tx, prize)
		if err != nil {
			return err
		}
		prize.Amt = (*models.Number)(paid)
	case "config":
		if err := prizeAddress(prize.Settler); err != nil {
			return fmt.Errorf("settler: %v", err)
		}
		err = e.dbc.PrizeConfig(tx, prize)
	case "close":
		err = e.dbc.PrizeClose(tx, prize)
	default:
		err = fmt.Errorf("unknown op %q", prize.Op)
	}
	if err != nil {
		return err
	}

	return tx.Model(&models.PrizeInfo{}).Where("tx_hash = ?", prize.TxHash).Updates(map[string]interface{}{
		"order_status": 0,
		"amt":          prize.Amt,
	}).Error
}

// prizePoolsAbove returns the pools that have a prize_info row above height. fork()
// calls it before delInfo drops those rows.
func prizePoolsAbove(tx *gorm.DB, height int64) ([]string, error) {
	var pools []string
	err := tx.Model(&models.PrizeInfo{}).Where("block_number > ?", height).Distinct("pool_id").Order("pool_id").Pluck("pool_id", &pools).Error
	if err != nil {
		return nil, fmt.Errorf("FindPrizePools error: %v", err)
	}
	return pools, nil
}

// prizeFork recomputes the pools touched above height from their remaining
// successful ops. delInfo has already dropped prize_info, prize_deposit and
// prize_settle_entry rows above height; drc20Fork reverts the reserve balances.
func (e *Explorer) prizeFork(tx *gorm.DB, pools []string) error {
	log.Info("fork", "prize_v1", len(pools))
	for _, pool := range pools {
		if err := e.dbc.PrizeRecompute(tx, pool); err != nil {
			return fmt.Errorf("PrizeRecompute %s error: %v", pool, err)
		}
	}
	return nil
}
