package explorer

import (
	"dogeuni-indexer/models"
	"fmt"

	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/btcutil"
	"github.com/dogecoinw/doged/chaincfg"
	"github.com/dogecoinw/doged/chaincfg/chainhash"
)

// reserveModels are the tables that record a protocol reserve address. Each reserve
// is P2SH(<string>) of a protocol-chosen string, which nobody holds a key for: only
// the indexer's own rules may move what it holds.
var reserveModels = []interface{}{
	&models.SwapLiquidity{},
	&models.SwapV2Liquidity{},
	&models.PumpLiquidity{},
	&models.StakeCollect{},
	&models.StakeV2Collect{},
	&models.BoxCollect{},
	&models.ExchangeCollect{},
	&models.FileExchangeCollect{},
	&models.ConsensusStakeRecord{},
}

// commitSender returns the address that authorises an inscription: the owner of the
// output that the commit tx's first input spends.
//
// From reserveSenderHeight on, that address must not be a protocol reserve. The
// redeem script of a reserve is the string itself, and some strings execute to true
// (swap's P2SH("WOOWOW") is OP_7 OP_1NEGATE ...), so anyone could spend a UTXO sent
// to the reserve as the commit input and move the reserve's tokens as its "owner".
func (e *Explorer) commitSender(commit *btcjson.TxRawResult, height int64) (string, error) {
	prevHash, _ := chainhash.NewHashFromStr(commit.Vin[0].Txid)
	prev, err := e.node.GetRawTransactionVerboseBool(prevHash)
	if err != nil {
		return "", fmt.Errorf("%w: %v", CHAIN_NETWORK_ERR, err)
	}

	sender, err := outputAddress(prev, int(commit.Vin[0].Vout))
	if err != nil {
		return "", err
	}

	if activated(e.reserveSenderHeight, height) {
		reserve, err := e.isReserve(sender)
		if err != nil {
			return "", err
		}
		if reserve {
			return "", fmt.Errorf("sender %s is a protocol reserve", sender)
		}
	}

	return sender, nil
}

// isReserve reports whether address is a protocol reserve. Reserves are all P2SH, so
// other addresses need no lookup.
func (e *Explorer) isReserve(address string) (bool, error) {
	addr, err := btcutil.DecodeAddress(address, &chaincfg.MainNetParams)
	if err != nil {
		return false, nil
	}
	if _, ok := addr.(*btcutil.AddressScriptHash); !ok {
		return false, nil
	}

	for _, m := range reserveModels {
		var n int64
		if err := e.dbc.DB.Model(m).Where("reserves_address = ?", address).Limit(1).Count(&n).Error; err != nil {
			return false, fmt.Errorf("%w: reserve lookup: %v", STORAGE_ERR, err)
		}
		if n > 0 {
			return true, nil
		}
	}
	return false, nil
}
