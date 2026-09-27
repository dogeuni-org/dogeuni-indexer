package router_v3

import (
	"bytes"
	"dogeuni-indexer/storage_v3"
	"dogeuni-indexer/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

const pairs = 120

// newPairRouter serves pairs pairs. Liquidity takes only 3 values, so most rows tie on
// the sort key, and every pair has an older summary row that must not be counted. The
// latest summaries are written in reverse tick order, so id order differs from the
// tick order the GROUP BY scan yields.
func newPairRouter(t *testing.T) *Router {
	db := storage_v3.NewSqliteClient(utils.SqliteConfig{Switch: true, Database: filepath.Join(t.TempDir(), "t.db")})
	if db == nil {
		t.Fatal("open sqlite")
	}
	t.Cleanup(func() { db.MysqlDB.Close() })
	exec := func(q string, args ...interface{}) {
		if _, err := db.MysqlDB.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`CREATE TABLE swap_summary_liquidity (id integer primary key, tick text, tick0 text, tick1 text,
		open_price real, close_price real, base_volume real, doge_usdt real, liquidity real)`)
	exec(`CREATE TABLE swap_liquidity (id integer primary key, tick text, amt0 text, amt1 text)`)
	for i := 0; i < pairs; i++ {
		tick := fmt.Sprintf("T%03d", i)
		exec(`INSERT INTO swap_summary_liquidity (tick, tick0, tick1, open_price, close_price, base_volume, doge_usdt, liquidity)
			VALUES (?, 'A', 'B', 1, 1, 0, 0, 999)`, tick)
	}
	for i := pairs - 1; i >= 0; i-- {
		tick := fmt.Sprintf("T%03d", i)
		exec(`INSERT INTO swap_summary_liquidity (tick, tick0, tick1, open_price, close_price, base_volume, doge_usdt, liquidity)
			VALUES (?, 'A', 'B', 1, 2, 0, 0, ?)`, tick, i%3)
		exec(`INSERT INTO swap_liquidity (tick, amt0, amt1) VALUES (?, '1', '2')`, tick)
	}
	return &Router{mysql: db}
}

type pairPage struct {
	Data  []utils.SwapPairSummary `json:"data"`
	Total int64                   `json:"total"`
}

func postPairs(t *testing.T, r *Router, body string) (int, pairPage) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v3/swap/pair/all", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	r.SwapPairAll(c)
	var p pairPage
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
	}
	return w.Code, p
}

func mustPairs(t *testing.T, r *Router, body string) pairPage {
	code, p := postPairs(t, r, body)
	if code != http.StatusOK {
		t.Fatalf("%q: status %d", body, code)
	}
	return p
}

// Without a limit every pair is returned, as before paging existed, now with the total.
func TestSwapPairAllUnpaged(t *testing.T) {
	r := newPairRouter(t)
	for _, body := range []string{``, `{}`, `{"offset":10}`, `{"limit":null}`} {
		p := mustPairs(t, r, body)
		if len(p.Data) != pairs || p.Total != pairs {
			t.Errorf("%q: %d rows total %d, want %d/%d", body, len(p.Data), p.Total, pairs, pairs)
		}
	}
	p := mustPairs(t, r, ``)
	for i := 1; i < len(p.Data); i++ {
		prev, cur := p.Data[i-1], p.Data[i]
		if cur.Liquidity > prev.Liquidity {
			t.Fatalf("row %d: liquidity %v after %v, want highest first", i, cur.Liquidity, prev.Liquidity)
		}
		// ties follow id, which here is reverse tick order
		if cur.Liquidity == prev.Liquidity && cur.Tick > prev.Tick {
			t.Fatalf("row %d: tie %s after %s, want id order", i, cur.Tick, prev.Tick)
		}
	}
	if p.Data[0].Liquidity != 4 || p.Data[0].PriceChangePercent24H != 100 {
		t.Fatalf("first row %+v is not from the latest summary", p.Data[0])
	}
}

// Walking the pages yields exactly the unpaged list: nothing repeated, nothing missed,
// although most rows tie on liquidity.
func TestSwapPairAllPages(t *testing.T) {
	r := newPairRouter(t)
	full := mustPairs(t, r, ``).Data
	var walked []utils.SwapPairSummary
	for offset := 0; ; offset += 7 {
		p := mustPairs(t, r, fmt.Sprintf(`{"limit":7,"offset":%d}`, offset))
		if p.Total != pairs {
			t.Fatalf("offset %d: total %d, want %d", offset, p.Total, pairs)
		}
		if len(p.Data) == 0 {
			break
		}
		walked = append(walked, p.Data...)
	}
	if len(walked) != len(full) {
		t.Fatalf("pages hold %d rows, want %d", len(walked), len(full))
	}
	for i := range full {
		if walked[i].Tick != full[i].Tick {
			t.Fatalf("row %d: page order %s, full order %s", i, walked[i].Tick, full[i].Tick)
		}
	}
}

// A given limit follows the router_v3 bounds: at most 50, negatives clamped.
func TestSwapPairAllBounds(t *testing.T) {
	r := newPairRouter(t)
	for _, c := range []struct {
		body string
		rows int
	}{
		{`{"limit":100000}`, 50},
		{`{"limit":-1}`, 50},
		{`{"limit":10,"offset":-5}`, 10},
		{`{"limit":0}`, 0},
		{`{"limit":50,"offset":100}`, pairs - 100},
		{`{"limit":10,"offset":100000}`, 0},
	} {
		p := mustPairs(t, r, c.body)
		if len(p.Data) != c.rows || p.Total != pairs {
			t.Errorf("%s: %d rows total %d, want %d/%d", c.body, len(p.Data), p.Total, c.rows, pairs)
		}
	}
	if code, _ := postPairs(t, r, `{"limit":"ten"}`); code != http.StatusBadRequest {
		t.Errorf("malformed body: status %d, want 400", code)
	}
}
