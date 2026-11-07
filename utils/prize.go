package utils

import (
	"dogeuni-indexer/models"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

const (
	PrizeMemoMaxLen = 64
	// a pool_id is a txid; anything longer cannot name a pool
	prizePoolIdMaxLen = 64
)

// ParseDecimal parses a non-negative decimal integer in canonical form: digits only,
// no sign, no leading zeros (except "0" itself).
func ParseDecimal(s string) (*big.Int, error) {
	if s == "" {
		return nil, fmt.Errorf("empty number")
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return nil, fmt.Errorf("number %q is not a decimal integer", s)
		}
	}
	if len(s) > 1 && s[0] == '0' {
		return nil, fmt.Errorf("number %q has leading zeros", s)
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("number %q is not a decimal integer", s)
	}
	return n, nil
}

func parseDecimalInt64(s string) (int64, error) {
	n, err := ParseDecimal(s)
	if err != nil {
		return 0, err
	}
	if !n.IsInt64() {
		return 0, fmt.Errorf("number %q out of range", s)
	}
	return n.Int64(), nil
}

// PrintableASCII reports whether s holds only bytes 0x20-0x7e.
func PrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// ParsePrizeEntries parses the [[address, cum], ...] JSON of a settle.
func ParsePrizeEntries(raw string) ([]models.PrizeEntry, error) {
	var pairs [][]string
	if err := json.Unmarshal([]byte(raw), &pairs); err != nil {
		return nil, fmt.Errorf("entries: %v", err)
	}
	return convertPrizeEntries(pairs)
}

func convertPrizeEntries(pairs [][]string) ([]models.PrizeEntry, error) {
	entries := make([]models.PrizeEntry, 0, len(pairs))
	for i, pair := range pairs {
		if len(pair) != 2 {
			return nil, fmt.Errorf("entry %d is not [address, cum]", i)
		}
		cum, err := ParseDecimal(pair[1])
		if err != nil {
			return nil, fmt.Errorf("entry %d cum: %v", i, err)
		}
		entries = append(entries, models.PrizeEntry{Address: pair[0], Cum: (*models.Number)(cum)})
	}
	return entries, nil
}

// ConvertPrize turns the inscription into an info row, parsing only the fields its
// op uses. Format errors reject the inscription before it is stored; the rules that
// depend on state are checked when it is executed.
func ConvertPrize(inscription *models.PrizeInscription) (*models.PrizeInfo, error) {
	info := &models.PrizeInfo{Op: inscription.Op, Amt: models.NewNumber(0)}

	if inscription.Op != "create" {
		if len(inscription.PoolId) > prizePoolIdMaxLen {
			return nil, fmt.Errorf("pool_id too long")
		}
		info.PoolId = inscription.PoolId
	}

	// pool ids, settlers and entry addresses are txids and base58 addresses; anything
	// outside printable ASCII is invalid, and refusing it before the row is stored keeps
	// MySQL (utf8 vs utf8mb4) and SQLite instances from disagreeing on the save
	if !PrintableASCII(inscription.PoolId) || !PrintableASCII(inscription.Settler) {
		return nil, fmt.Errorf("pool_id and settler must be printable ASCII")
	}
	for i, pair := range inscription.Entries {
		if len(pair) > 0 && !PrintableASCII(pair[0]) {
			return nil, fmt.Errorf("entry %d address is not printable ASCII", i)
		}
	}

	var err error
	switch inscription.Op {
	case "create":
		info.Tick = strings.ToUpper(inscription.Tick)
		info.Settler = inscription.Settler
		if info.Timeout, err = parseDecimalInt64(inscription.Timeout); err != nil {
			return nil, fmt.Errorf("timeout: %v", err)
		}
	case "deposit":
		amt, err := ParseDecimal(inscription.Amt)
		if err != nil {
			return nil, fmt.Errorf("amt: %v", err)
		}
		info.Amt = (*models.Number)(amt)
		// checked before the row is stored: a non-ASCII memo is what would make
		// MySQL (utf8 vs utf8mb4) and SQLite instances disagree
		if len(inscription.Memo) > PrizeMemoMaxLen {
			return nil, fmt.Errorf("memo longer than %d bytes", PrizeMemoMaxLen)
		}
		if !PrintableASCII(inscription.Memo) {
			return nil, fmt.Errorf("memo is not printable ASCII")
		}
		info.Memo = inscription.Memo
	case "settle":
		if info.Seq, err = parseDecimalInt64(inscription.Seq); err != nil {
			return nil, fmt.Errorf("seq: %v", err)
		}
		if info.Upto, err = parseDecimalInt64(inscription.Upto); err != nil {
			return nil, fmt.Errorf("upto: %v", err)
		}
		if _, err := convertPrizeEntries(inscription.Entries); err != nil {
			return nil, err
		}
		if inscription.Entries == nil {
			inscription.Entries = [][]string{}
		}
		raw, err := json.Marshal(inscription.Entries)
		if err != nil {
			return nil, err
		}
		info.Entries = string(raw)
	case "config":
		info.Settler = inscription.Settler
	case "claim", "refund", "withdraw", "close":
	default:
		return nil, fmt.Errorf("unknown op %q", inscription.Op)
	}
	return info, nil
}
