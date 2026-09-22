package utils

import (
	"encoding/json"
	"fmt"
	"testing"
)

// nodeValue decodes sats the way a vout value arrives from the node: as a JSON
// number with 8 decimals.
func nodeValue(t *testing.T, sats int64) float64 {
	var v float64
	if err := json.Unmarshal([]byte(fmt.Sprintf("%d.%08d", sats/SatoshisPerDoge, sats%SatoshisPerDoge)), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestSatoshisExact(t *testing.T) {
	check := func(sats int64) {
		if got := Satoshis(nodeValue(t, sats)); got != sats {
			t.Fatalf("Satoshis(%d sats) = %d", sats, got)
		}
	}
	for sats := int64(0); sats <= 2_000_000; sats++ {
		check(sats)
	}
	// every satoshi is representable up to 2^53 sats (about 90M DOGE)
	for sats := int64(2_000_000); sats < 1<<53; sats = sats*3 + 7 {
		check(sats)
	}
}
