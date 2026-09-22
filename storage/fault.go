package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/go-sql-driver/mysql"
	"github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
)

// FaultRecorder keeps the first database fault seen by the statements of one
// DBClient. A fault is a failure of the database itself (connection, lock, disk),
// as opposed to an error caused by the data being written, which every indexer
// reproduces identically. Callers that turn errors into strings still let the
// scanner tell the two apart through the recorder.
type FaultRecorder struct {
	mu  sync.Mutex
	err error
}

func (r *FaultRecorder) observe(err error) {
	if err == nil || !IsFault(err) {
		return
	}
	r.mu.Lock()
	if r.err == nil {
		r.err = err
	}
	r.mu.Unlock()
}

// Take returns the recorded fault, if any, and clears it.
func (r *FaultRecorder) Take() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	err := r.err
	r.err = nil
	return err
}

// Faults that retrying may cure, or that an operator must fix before the index can
// continue. Anything else — constraint violations, values out of range, strings too
// long — follows from the data and is not a fault: retrying it would stall the
// indexer on a tx anyone can broadcast.
var (
	mysqlFaults = map[uint16]bool{
		1021: true, // disk full
		1030: true, // error from storage engine
		1040: true, // too many connections
		1053: true, // server shutdown in progress
		1114: true, // table is full
		1180: true, // error during commit
		1203: true, // max_user_connections
		1205: true, // lock wait timeout
		1213: true, // deadlock
		1226: true, // user resource limit
		1290: true, // --read-only
		1317: true, // query interrupted
		1792: true, // read-only transaction
		1836: true, // read-only mode
		1927: true, // connection killed
		2006: true, // server has gone away
		2013: true, // lost connection
		3024: true, // max_execution_time exceeded
	}
	sqliteFaults = map[sqlite3.ErrNo]bool{
		sqlite3.ErrNomem:     true,
		sqlite3.ErrReadonly:  true,
		sqlite3.ErrInterrupt: true,
		sqlite3.ErrIoErr:     true,
		sqlite3.ErrCorrupt:   true,
		sqlite3.ErrFull:      true,
		sqlite3.ErrCantOpen:  true,
		sqlite3.ErrProtocol:  true,
		sqlite3.ErrNotADB:    true,
		sqlite3.ErrBusy:      true,
		sqlite3.ErrLocked:    true,
	}
)

// IsFault reports whether err is a failure of the database rather than of the data.
func IsFault(err error) bool {
	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return mysqlFaults[myErr.Number]
	}
	var liteErr sqlite3.Error
	if errors.As(err, &liteErr) {
		return sqliteFaults[liteErr.Code]
	}
	var netErr net.Error
	return errors.As(err, &netErr) ||
		errors.Is(err, driver.ErrBadConn) ||
		errors.Is(err, mysql.ErrInvalidConn) ||
		errors.Is(err, mysql.ErrMalformPkt) ||
		errors.Is(err, mysql.ErrPktSync) ||
		errors.Is(err, mysql.ErrPktSyncMul) ||
		errors.Is(err, mysql.ErrBusyBuffer) ||
		errors.Is(err, sql.ErrConnDone) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

// WithFaultRecorder returns a client sharing c's connections whose statements,
// transaction begins and commits report faults to the returned recorder.
func (c *DBClient) WithFaultRecorder() (*DBClient, *FaultRecorder) {
	registerFaultCallbacks(c.DB)
	rec := &FaultRecorder{}
	// A Context forces a private Statement, so the pool swap below stays local.
	db := c.DB.Session(&gorm.Session{NewDB: true, Context: context.Background()})
	db.Statement.ConnPool = &faultPool{ConnPool: db.Statement.ConnPool, rec: rec}
	return &DBClient{DB: db, lock: c.lock}, rec
}

type faultSink interface{ recorder() *FaultRecorder }

// faultPool sees transaction begins; faultTx sees commits. Statement errors are
// seen by the callbacks, which run after gorm has read all rows.
type faultPool struct {
	gorm.ConnPool
	rec *FaultRecorder
}

func (p *faultPool) recorder() *FaultRecorder { return p.rec }

func (p *faultPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.ConnPool.(gorm.TxBeginner).BeginTx(ctx, opts)
	if err != nil {
		p.rec.observe(err)
		return nil, err
	}
	return &faultTx{Tx: tx, rec: p.rec}, nil
}

func (p *faultPool) GetDBConn() (*sql.DB, error) {
	return p.ConnPool.(*sql.DB), nil
}

type faultTx struct {
	*sql.Tx
	rec *FaultRecorder
}

func (t *faultTx) recorder() *FaultRecorder { return t.rec }

func (t *faultTx) Commit() error {
	err := t.Tx.Commit()
	t.rec.observe(err)
	return err
}

const faultCallback = "dogeuni:record_fault"

var registerMu sync.Mutex

func registerFaultCallbacks(db *gorm.DB) {
	registerMu.Lock()
	defer registerMu.Unlock()
	if db.Callback().Query().Get(faultCallback) != nil {
		return
	}
	record := func(db *gorm.DB) {
		if sink, ok := db.Statement.ConnPool.(faultSink); ok {
			sink.recorder().observe(db.Error)
		}
	}
	cb := db.Callback()
	// Last in each chain, so the default transaction's commit is included.
	_ = cb.Create().After("gorm:commit_or_rollback_transaction").Register(faultCallback, record)
	_ = cb.Update().After("gorm:commit_or_rollback_transaction").Register(faultCallback, record)
	_ = cb.Delete().After("gorm:commit_or_rollback_transaction").Register(faultCallback, record)
	_ = cb.Query().After("gorm:after_query").Register(faultCallback, record)
	_ = cb.Row().After("gorm:row").Register(faultCallback, record)
	_ = cb.Raw().After("gorm:raw").Register(faultCallback, record)
}
