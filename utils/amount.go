package utils

import "math"

// SatoshisPerDoge is the number of satoshis in one DOGE.
const SatoshisPerDoge = 100000000

// Satoshis converts an output value in DOGE, as the node reports it in JSON, to
// satoshis. The node prints exactly 8 decimals and the decoded float64 is the one
// nearest to that value, so rounding recovers the satoshi amount exactly.
func Satoshis(doge float64) int64 {
	return int64(math.Round(doge * SatoshisPerDoge))
}
