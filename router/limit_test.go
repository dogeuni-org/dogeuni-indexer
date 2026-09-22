package router

import (
	"bytes"
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestDB(t *testing.T, tables ...interface{}) *storage.DBClient {
	dbc := storage.NewSqliteClient(utils.SqliteConfig{Switch: true, Database: filepath.Join(t.TempDir(), "t.db")})
	if err := dbc.DB.AutoMigrate(tables...); err != nil {
		t.Fatal(err)
	}
	return dbc
}

type page struct {
	Data  []json.RawMessage `json:"data"`
	Total int64             `json:"total"`
}

func post(t *testing.T, h gin.HandlerFunc, body string) page {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h(c)
	if w.Code != http.StatusOK {
		t.Fatalf("%s: status %d: %s", body, w.Code, w.Body.String())
	}
	var p page
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

// A client limit must not reach the database unbounded: negative means "no limit" to gorm.
func TestOrderLimitBounded(t *testing.T) {
	dbc := newTestDB(t, &models.WDogeInfo{})
	for i := 0; i < utils.MaxPageLimit+20; i++ {
		if err := dbc.DB.Create(&models.WDogeInfo{TxHash: fmt.Sprint(i)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	r := NewWdogeRouter(dbc, nil)
	for _, body := range []string{`{"limit":-1}`, `{"limit":100000}`, `{"limit":-1,"offset":-5}`} {
		if got := len(post(t, r.Order, body).Data); got != utils.MaxPageLimit {
			t.Errorf("%s: %d rows, want %d", body, got, utils.MaxPageLimit)
		}
	}
	if got := len(post(t, r.Order, `{"limit":7}`).Data); got != 7 {
		t.Errorf("limit 7: %d rows", got)
	}
}

// Every tick is still served in all-ticks mode, and an ordinary page never fills
// the all-ticks cache.
func TestDrc20CollectAllTicks(t *testing.T) {
	dbc := newTestDB(t, &models.Block{}, &models.Drc20Collect{}, &models.Drc20CollectAddress{})
	const n = utils.MaxPageLimit + 150
	for i := 0; i < n; i++ {
		if err := dbc.DB.Create(&models.Drc20Collect{Tick: fmt.Sprintf("T%03d", i), TxHash: fmt.Sprint(i)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	cacheDrc20CollectAll = nil
	r := NewDrc20Router(dbc, nil, nil, nil)

	if got := len(post(t, r.Collect, `{"limit":150}`).Data); got != utils.MaxPageLimit {
		t.Errorf("page: %d rows, want %d", got, utils.MaxPageLimit)
	}
	for i := 0; i < 2; i++ { // second call is served from the cache
		p := post(t, r.Collect, `{"limit":1000}`)
		if len(p.Data) != n || p.Total != n {
			t.Errorf("all ticks, call %d: %d rows, total %d, want %d", i, len(p.Data), p.Total, n)
		}
	}
	if got := len(post(t, r.Collect, `{"limit":10}`).Data); got != 10 {
		t.Errorf("page after cache: %d rows", got)
	}
	if p := post(t, r.Collect, `{"limit":1000}`); len(p.Data) != n {
		t.Errorf("cache poisoned by a page: %d rows, want %d", len(p.Data), n)
	}
}
