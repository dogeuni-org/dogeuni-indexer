package explorer

import (
	"dogeuni-indexer/utils"
	"math/big"

	"github.com/dogecoinw/doged/btcjson"
)

// A drc-20 mint pays 0.001 DOGE per repeat to output 0, up to MintMaxRepeat repeats.
const (
	MintUnitSats  = 100000
	MintMaxRepeat = 30
)

// voutSats is the value of output i in satoshis. From amount_height on it is exact.
// Before, it keeps the legacy conversion, which rounds float64 error up and so can
// read one satoshi more than the output holds; blocks already indexed that way must
// not change.
func (e *Explorer) voutSats(tx *btcjson.TxRawResult, i int, height int64) *big.Int {
	if activated(e.amountHeight, height) {
		return big.NewInt(utils.Satoshis(tx.Vout[i].Value))
	}
	return utils.Float64ToBigInt(tx.Vout[i].Value * utils.SatoshisPerDoge)
}

// mintRepeat returns the repeat count paid by a mint output of value DOGE, and
// whether value is exactly that many units. Before amount_height it keeps the legacy
// float arithmetic, under which honest mints of 9, 13, 18 and 26 repeats fail.
func (e *Explorer) mintRepeat(value float64, height int64) (int64, bool) {
	if activated(e.amountHeight, height) {
		sats := utils.Satoshis(value)
		repeat := min(sats/MintUnitSats, MintMaxRepeat)
		return repeat, sats == repeat*MintUnitSats
	}
	repeat := min(int64(value/0.001), MintMaxRepeat)
	return repeat, value == 0.001*float64(repeat)
}
