package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIsFault(t *testing.T) {
	cases := []struct {
		err   error
		fault bool
	}{
		{gorm.ErrRecordNotFound, false},
		{fmt.Errorf("the contract does not exist"), false},
		{&mysql.MySQLError{Number: 1406}, false}, // data too long: anyone can broadcast it
		{&mysql.MySQLError{Number: 1062}, false}, // duplicate key
		{&mysql.MySQLError{Number: 1213}, true},  // deadlock
		{fmt.Errorf("wrapped: %w", &mysql.MySQLError{Number: 1205}), true},
		{sqlite3.Error{Code: sqlite3.ErrConstraint}, false},
		{sqlite3.Error{Code: sqlite3.ErrTooBig}, false},
		{sqlite3.Error{Code: sqlite3.ErrBusy}, true},
		{sqlite3.Error{Code: sqlite3.ErrIoErr}, true},
		{driver.ErrBadConn, true},
		{mysql.ErrInvalidConn, true},
	}
	for _, c := range cases {
		if got := IsFault(c.err); got != c.fault {
			t.Errorf("IsFault(%v) = %v, want %v", c.err, got, c.fault)
		}
	}
}

// A commit that fails because another connection holds the database is a fault even
// though no statement failed. Rollback-journal mode makes the commit itself wait
// for readers.
func TestCommitFaultRecorded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	db, err := gorm.Open(sqlite.Open(path+"?_busy_timeout=50"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE t (v INTEGER)").Error; err != nil {
		t.Fatal(err)
	}
	c, rec := (&DBClient{DB: db}).WithFaultRecorder()

	reader, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	ctx := context.Background()
	conn, err := reader.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := conn.QueryRowContext(ctx, "SELECT count(*) FROM t").Scan(&n); err != nil {
		t.Fatal(err)
	}

	tx := c.DB.Begin()
	if err := tx.Exec("INSERT INTO t VALUES (1)").Error; err != nil {
		t.Fatal(err)
	}
	if rec.Take() != nil {
		t.Fatal("fault recorded before commit")
	}
	if err := tx.Commit().Error; err == nil {
		t.Fatal("commit succeeded while a reader held the database")
	}
	tx.Rollback()
	if fault := rec.Take(); !IsFault(fault) {
		t.Fatalf("commit fault not recorded, got %v", fault)
	}
	conn.ExecContext(ctx, "ROLLBACK")
}
