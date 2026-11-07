package explorer

import (
	"bytes"
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"

	"github.com/dogecoinw/doged/btcjson"
	"github.com/dogecoinw/doged/btcutil"
	"github.com/dogecoinw/doged/chaincfg"
	"github.com/dogecoinw/doged/txscript"
	"gorm.io/gorm"
)

const prizeTick = "PRZ"

var prizeTables = []interface{}{&models.PrizeInfo{}, &models.PrizePool{}, &models.PrizeEntitlement{}, &models.PrizeDeposit{}, &models.PrizeSettleEntry{}}

// every table that fork() touches, so tests can run the real fork
var allTables = []interface{}{
	&models.BoxInfo{}, &models.BoxCollect{}, &models.BoxCollectAddress{}, &models.BoxRevert{},
	&models.ConsensusInfo{}, &models.ConsensusRevert{}, &models.ConsensusStakeRecord{},
	&models.CrossInfo{}, &models.CrossCollect{}, &models.CrossRevert{}, &models.CrossBotInfo{},
	&models.Drc20Info{}, &models.Drc20Collect{}, &models.Drc20CollectAddress{}, &models.Drc20Revert{},
	&models.ExchangeInfo{}, &models.ExchangeCollect{}, &models.ExchangeRevert{}, &models.ExchangeSummary{},
	&models.FileExchangeInfo{}, &models.FileExchangeCollect{}, &models.FileExchangeRevert{}, &models.FileExchangeSummary{},
	&models.InviteInfo{}, &models.InviteCollect{}, &models.InviteRevert{},
	&models.FileInfo{}, &models.FileCollectAddress{}, &models.FileRevert{},
	&models.NftInfo{}, &models.NftCollect{}, &models.NftCollectAddress{}, &models.NftRevert{},
	&models.StakeInfo{}, &models.StakeCollect{}, &models.StakeCollectAddress{}, &models.StakeCollectReward{}, &models.StakeRevert{}, &models.StakeRewardInfo{}, &models.StakeRewardRevert{},
	&models.Meme20Info{}, &models.Meme20Collect{}, &models.Meme20CollectAddress{}, &models.Meme20Revert{},
	&models.StakeV2Info{}, &models.StakeV2Collect{}, &models.StakeV2CollectAddress{}, &models.StakeV2Revert{},
	&models.SwapV2Info{}, &models.SwapV2Liquidity{}, &models.SwapV2Revert{},
	&models.WDogeInfo{},
	&models.PumpInfo{}, &models.PumpLiquidity{}, &models.PumpRevert{}, &models.PumpInviteReward{}, &models.PumpInviteRewardRevert{},
	&models.SwapInfo{}, &models.SwapLiquidity{}, &models.SwapRevert{},
	&models.PrizeInfo{}, &models.PrizePool{}, &models.PrizeEntitlement{}, &models.PrizeDeposit{}, &models.PrizeSettleEntry{},
}

// paddr is a valid mainnet P2PKH address.
func paddr(i byte) string {
	a, err := btcutil.NewAddressPubKeyHash(bytes.Repeat([]byte{i}, 20), &chaincfg.MainNetParams)
	if err != nil {
		panic(err)
	}
	return a.EncodeAddress()
}

var (
	pCreator  = paddr(1)
	pSettler  = paddr(2)
	pSettler2 = paddr(3)
	pAlice    = paddr(4)
	pBob      = paddr(5)
	pDonor    = paddr(6)
	pW1       = paddr(11)
	pW2       = paddr(12)
	pW3       = paddr(13)
)

type prizeEnv struct {
	t *testing.T
	e *Explorer
	n int
}

func newPrizeEnv(t *testing.T, tables ...interface{}) *prizeEnv {
	if len(tables) == 0 {
		tables = append(append(append([]interface{}{}, drc20Tables...), prizeTables...), reserveModels...)
	}
	e := newTestExplorer(t, tables...)
	deployTick(t, e, prizeTick)
	p := &prizeEnv{t: t, e: e}
	for _, a := range []string{pCreator, pAlice, pBob, pDonor} {
		p.tx(func(tx *gorm.DB) error {
			return e.dbc.MintDrc20(tx, prizeTick, a, big.NewInt(1000), "mint-"+a, 1, false)
		})
	}
	return p
}

func (p *prizeEnv) tx(f func(tx *gorm.DB) error) {
	p.t.Helper()
	tx := p.e.dbc.DB.Begin()
	if err := f(tx); err != nil {
		tx.Rollback()
		p.t.Fatal(err)
	}
	if err := tx.Commit().Error; err != nil {
		p.t.Fatal(err)
	}
}

// op runs one inscription the way scanTx does after decode: save the pending row,
// execute, record a rejection. It returns the txid and the execute error.
func (p *prizeEnv) op(height int64, holder, js string) (string, error) {
	p.t.Helper()
	p.n++
	txid := fmt.Sprintf("%064x", p.n)
	param := &models.PrizeInscription{}
	if err := json.Unmarshal([]byte(js), param); err != nil {
		p.t.Fatal(err)
	}
	info, err := utils.ConvertPrize(param)
	if err != nil {
		p.t.Fatalf("convert %s: %v", js, err)
	}
	info.TxHash = txid
	info.BlockNumber = height
	info.HolderAddress = holder
	info.OrderStatus = 1
	if info.Op == "create" {
		info.PoolId = txid
	}
	if err := p.e.dbc.DB.Save(info).Error; err != nil {
		p.t.Fatal(err)
	}
	err = p.e.executePrize(info)
	if err != nil {
		if abort := p.e.failTx(&models.PrizeInfo{}, txid, err); abort != nil {
			p.t.Fatalf("op aborted the block: %v", abort)
		}
	}
	return txid, err
}

func (p *prizeEnv) ok(height int64, holder, js string) string {
	p.t.Helper()
	txid, err := p.op(height, holder, js)
	if err != nil {
		p.t.Fatalf("h%d %s: %v", height, js, err)
	}
	return txid
}

// fail runs an op that must be rejected and checks it changed nothing but its own
// prize_info row, which gets order_status 1 and err_info.
func (p *prizeEnv) fail(height int64, holder, js, want string) {
	p.t.Helper()
	before := prizeState(p.t, p.e)
	txid, err := p.op(height, holder, js)
	if err == nil {
		p.t.Fatalf("h%d %s: want failure %q, got success", height, js, want)
	}
	if !strings.Contains(err.Error(), want) {
		p.t.Fatalf("h%d %s: want failure %q, got %v", height, js, want, err)
	}
	if after := prizeState(p.t, p.e); after != before {
		p.t.Fatalf("failed op changed state:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	row := &models.PrizeInfo{}
	p.e.dbc.DB.Where("tx_hash = ?", txid).First(row)
	if row.OrderStatus != 1 || row.ErrInfo == nil || *row.ErrInfo == "" {
		p.t.Fatalf("failed op row: order_status=%d err_info=%v", row.OrderStatus, row.ErrInfo)
	}
}

func (p *prizeEnv) create(height int64, timeout int) string {
	return p.ok(height, pCreator, fmt.Sprintf(`{"p":"prize-v1","op":"create","tick":"prz","settler":%q,"timeout":"%d"}`, pSettler, timeout))
}

func deposit(pool string, amt int) string {
	return fmt.Sprintf(`{"p":"prize-v1","op":"deposit","pool_id":%q,"amt":"%d"}`, pool, amt)
}

func settle(pool string, seq, upto int64, entries ...interface{}) string {
	pairs := [][]string{}
	for i := 0; i+1 < len(entries); i += 2 {
		pairs = append(pairs, []string{entries[i].(string), fmt.Sprint(entries[i+1])})
	}
	raw, _ := json.Marshal(pairs)
	return fmt.Sprintf(`{"p":"prize-v1","op":"settle","pool_id":%q,"seq":"%d","upto":"%d","entries":%s}`, pool, seq, upto, raw)
}

func poolOp(op, pool string) string {
	return fmt.Sprintf(`{"p":"prize-v1","op":%q,"pool_id":%q}`, op, pool)
}

func configOp(pool, settler string) string {
	return fmt.Sprintf(`{"p":"prize-v1","op":"config","pool_id":%q,"settler":%q}`, pool, settler)
}

func (p *prizeEnv) pool(id string) *models.PrizePool {
	p.t.Helper()
	pool := &models.PrizePool{}
	if err := p.e.dbc.DB.Where("pool_id = ?", id).First(pool).Error; err != nil {
		p.t.Fatalf("pool %s: %v", id, err)
	}
	return pool
}

func (p *prizeEnv) balance(addr string) int64 {
	p.t.Helper()
	b, err := storage.PrizeBalance(p.e.dbc.DB, prizeTick, addr)
	if err != nil {
		p.t.Fatal(err)
	}
	return b.Int64()
}

func (p *prizeEnv) reserve(pool string) string {
	r, err := storage.PrizeReserve(pool)
	if err != nil {
		p.t.Fatal(err)
	}
	return r
}

func (p *prizeEnv) ent(pool, addr string) (cum, claimed int64) {
	ent := &models.PrizeEntitlement{}
	if err := p.e.dbc.DB.Where("pool_id = ? AND address = ?", pool, addr).First(ent).Error; err != nil {
		return -1, -1
	}
	return ent.Cum.Int64(), ent.Claimed.Int64()
}

func (p *prizeEnv) paid(txid string) int64 {
	row := &models.PrizeInfo{}
	p.e.dbc.DB.Where("tx_hash = ?", txid).First(row)
	return row.Amt.Int64()
}

// prizeState renders the prize tables and the non-zero DRC-20 balances without ids
// or timestamps, so two explorers can be compared.
func prizeState(t *testing.T, e *Explorer) string {
	t.Helper()
	db := e.dbc.DB
	var lines []string
	var pools []models.PrizePool
	if err := db.Order("pool_id").Find(&pools).Error; err != nil {
		t.Fatal(err)
	}
	for _, p := range pools {
		lines = append(lines, fmt.Sprintf("pool %s tick=%s creator=%s settler=%s timeout=%d reserve=%s seq=%d upto=%d last=%d owed=%s status=%s h=%d",
			p.PoolId, p.Tick, p.Creator, p.Settler, p.Timeout, p.ReservesAddress, p.Seq, p.Upto, p.LastSettleHeight, p.Owed, p.Status, p.BlockNumber))
	}
	var ents []models.PrizeEntitlement
	db.Order("pool_id, address").Find(&ents)
	for _, x := range ents {
		lines = append(lines, fmt.Sprintf("ent %s %s cum=%s claimed=%s", x.PoolId, x.Address, x.Cum, x.Claimed))
	}
	var deps []models.PrizeDeposit
	db.Order("pool_id, tx_hash").Find(&deps)
	for _, d := range deps {
		lines = append(lines, fmt.Sprintf("dep %s %s %s amt=%s memo=%q h=%d refunded=%v", d.PoolId, d.TxHash, d.HolderAddress, d.Amt, d.Memo, d.BlockNumber, d.Refunded))
	}
	var entries []models.PrizeSettleEntry
	db.Order("pool_id, tx_hash, address").Find(&entries)
	for _, x := range entries {
		lines = append(lines, fmt.Sprintf("entry %s %s %s cum=%s h=%d", x.PoolId, x.TxHash, x.Address, x.Cum, x.BlockNumber))
	}
	var bals []models.Drc20CollectAddress
	db.Find(&bals)
	var balLines []string
	for _, b := range bals {
		if b.AmtSum.Int().Sign() != 0 {
			balLines = append(balLines, fmt.Sprintf("bal %s %s %s", b.Tick, b.HolderAddress, b.AmtSum))
		}
	}
	sort.Strings(balLines)
	return strings.Join(append(lines, balLines...), "\n")
}

func prizeInfos(t *testing.T, e *Explorer) string {
	var rows []models.PrizeInfo
	if err := e.dbc.DB.Order("tx_hash").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, r := range rows {
		lines = append(lines, fmt.Sprintf("%s %s %s status=%d amt=%s err=%v", r.TxHash, r.Op, r.PoolId, r.OrderStatus, r.Amt, r.ErrInfo != nil))
	}
	return strings.Join(lines, "\n")
}

func TestPrizeHappyPath(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	reserve := p.reserve(pool)

	if got := p.pool(pool); got.Tick != prizeTick || got.Creator != pCreator || got.Settler != pSettler || got.Seq != 0 ||
		got.Upto != 9 || got.LastSettleHeight != 10 || got.Owed.Int64() != 0 || got.Status != models.PrizeStatusOpen || got.ReservesAddress != reserve {
		t.Fatalf("created pool: %+v", got)
	}

	p.ok(10, pAlice, deposit(pool, 100)) // same block as create: still pending
	p.ok(12, pBob, fmt.Sprintf(`{"p":"prize-v1","op":"deposit","pool_id":%q,"amt":"50","memo":"round 7"}`, pool))
	if p.balance(reserve) != 150 || p.balance(pAlice) != 900 || p.balance(pBob) != 950 {
		t.Fatalf("after deposits: reserve=%d alice=%d bob=%d", p.balance(reserve), p.balance(pAlice), p.balance(pBob))
	}

	p.ok(13, pSettler, settle(pool, 1, 12, pW1, 120, pW2, 30))
	if got := p.pool(pool); got.Seq != 1 || got.Upto != 12 || got.LastSettleHeight != 13 || got.Owed.Int64() != 150 {
		t.Fatalf("after settle: %+v", got)
	}

	claim := p.ok(14, pW1, poolOp("claim", pool))
	if p.balance(pW1) != 120 || p.paid(claim) != 120 || p.balance(reserve) != 30 {
		t.Fatalf("claim: w1=%d paid=%d reserve=%d", p.balance(pW1), p.paid(claim), p.balance(reserve))
	}
	if cum, claimed := p.ent(pool, pW1); cum != 120 || claimed != 120 {
		t.Fatalf("w1 entitlement %d/%d", cum, claimed)
	}
	p.fail(15, pW1, poolOp("claim", pool), "nothing to claim")
	p.fail(15, pW3, poolOp("claim", pool), "nothing to claim")

	p.ok(15, pW2, poolOp("claim", pool))
	if got := p.pool(pool); got.Owed.Int64() != 0 || p.balance(reserve) != 0 || p.balance(pW2) != 30 {
		t.Fatalf("after claims: owed=%s reserve=%d", got.Owed, p.balance(reserve))
	}
}

// settle may only allocate what is settled; pending deposits stay refundable
func TestPrizeSettleInvariant(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	reserve := p.reserve(pool)
	p.ok(11, pAlice, deposit(pool, 100))

	p.fail(12, pSettler, settle(pool, 1, 11, pW1, 101), "exceeds balance")
	// leaving Alice's deposit pending, nothing at all can be allocated
	p.fail(12, pSettler, settle(pool, 1, 10, pW1, 1), "exceeds balance")

	// a plain transfer into the reserve is allocatable, but is no deposit
	p.tx(func(tx *gorm.DB) error {
		return p.e.dbc.TransferDrc20(tx, prizeTick, pDonor, reserve, big.NewInt(1), "donate", 12, false)
	})
	p.ok(13, pSettler, settle(pool, 1, 10, pW1, 1))
	p.fail(14, pSettler, settle(pool, 2, 10, pW1, 2), "exceeds balance")
	p.fail(14, pDonor, poolOp("refund", pool), "not closed")

	// Alice's 100 is still pending and refundable after expiry
	p.ok(11+1440+1, pAlice, poolOp("refund", pool))
	p.fail(11+1440+1, pDonor, poolOp("refund", pool), "no pending deposit")
	if p.balance(pAlice) != 1000 || p.balance(reserve) != 1 {
		t.Fatalf("refund: alice=%d reserve=%d", p.balance(pAlice), p.balance(reserve))
	}
	p.ok(11+1440+2, pW1, poolOp("claim", pool))
	if p.balance(reserve) != 0 || p.balance(pW1) != 1 {
		t.Fatalf("claim after refund: reserve=%d w1=%d", p.balance(reserve), p.balance(pW1))
	}
}

func TestPrizeSettleChecks(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	reserve := p.reserve(pool)
	p.ok(11, pAlice, deposit(pool, 100))
	p.ok(12, pSettler, settle(pool, 1, 11, pW1, 50, pW2, 10))

	p.fail(13, pSettler, settle(pool, 1, 12, pW1, 50), "seq")
	p.fail(13, pSettler, settle(pool, 3, 12, pW1, 50), "seq")
	p.fail(13, pSettler, settle(pool, 2, 10, pW1, 50), "upto")
	p.fail(13, pSettler, settle(pool, 2, 13, pW1, 50), "upto")
	p.fail(13, pSettler, settle(pool, 2, 12, pW1, 49), "decreases")
	p.fail(13, pSettler, settle(pool, 2, 12, pW3, 1, pW3, 2), "repeated")
	p.fail(13, pSettler, settle(pool, 2, 12, reserve, 1), "reserve")
	p.fail(13, pSettler, settle(pool, 2, 12, "DNotAnAddress", 1), "address")
	// a compressed public key decodes, but is not the address form
	p.fail(13, pSettler, settle(pool, 2, 12, "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798", 1), "P2PKH")
	p.fail(13, pSettler, settle(pool, 2, 12), "entries")
	var many []interface{}
	for i := 0; i < 51; i++ {
		many = append(many, paddr(byte(100+i)), 0)
	}
	p.fail(13, pSettler, settle(pool, 2, 12, many...), "entries")
	p.fail(13, pCreator, settle(pool, 2, 12, pW1, 50), "not the settler")
	p.fail(13, pSettler, settle(fmt.Sprintf("%064x", 999), 1, 12, pW1, 50), "not found")

	// 50 entries is the limit
	p.ok(13, pSettler, settle(pool, 2, 12, many[:100]...))

	// the same cumulative amounts again: no effect beyond seq / upto
	before := p.pool(pool).Owed.Int64()
	p.ok(14, pSettler, settle(pool, 3, 12, pW1, 50, pW2, 10))
	if got := p.pool(pool); got.Owed.Int64() != before || got.Seq != 3 {
		t.Fatalf("resend: owed %s want %d, seq %d", got.Owed, before, got.Seq)
	}
	if cum, claimed := p.ent(pool, pW1); cum != 50 || claimed != 0 {
		t.Fatalf("resend: w1 %d/%d", cum, claimed)
	}
	p.ok(15, pW1, poolOp("claim", pool))
	p.ok(16, pSettler, settle(pool, 4, 12, pW1, 50))
	p.fail(17, pW1, poolOp("claim", pool), "nothing to claim")
}

func TestPrizeExpiry(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	p.ok(20, pAlice, deposit(pool, 100))

	p.fail(20+1440, pAlice, poolOp("refund", pool), "not closed")
	p.ok(20+1440, pBob, deposit(pool, 40)) // not yet expired
	p.ok(20+1441, pAlice, poolOp("refund", pool))
	if got := p.pool(pool); got.Status != models.PrizeStatusClosed {
		t.Fatalf("refund did not close the pool: %s", got.Status)
	}
	if p.balance(pAlice) != 1000 {
		t.Fatalf("alice=%d", p.balance(pAlice))
	}
	p.fail(20+1442, pSettler, settle(pool, 1, 20+1441, pW1, 1), "closed")
	p.fail(20+1442, pAlice, deposit(pool, 1), "closed")
	p.fail(20+1442, pCreator, configOp(pool, pSettler2), "closed")
	p.fail(20+1442, pCreator, poolOp("close", pool), "closed")
	// Bob's later deposit is refundable too
	p.ok(20+1442, pBob, poolOp("refund", pool))
	p.fail(20+1443, pBob, poolOp("refund", pool), "no pending deposit")
	if p.balance(pBob) != 1000 || p.balance(p.reserve(pool)) != 0 {
		t.Fatalf("bob=%d reserve=%d", p.balance(pBob), p.balance(p.reserve(pool)))
	}
}

// re-settling without advancing upto refreshes last_settle_height but not expiry
func TestPrizeSettlerCannotBlockExpiry(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	p.ok(20, pAlice, deposit(pool, 100))
	for i, h := range []int64{100, 800, 1400, 1460} {
		p.ok(h, pSettler, settle(pool, int64(i+1), 9, pW1, 0))
	}
	if got := p.pool(pool); got.LastSettleHeight != 1460 || got.Upto != 9 {
		t.Fatalf("pool: %+v", got)
	}
	p.fail(1461, pSettler, settle(pool, 5, 9, pW1, 0), "closed")
	p.ok(1461, pAlice, poolOp("refund", pool))
}

func TestPrizeNoPendingNeverExpires(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	p.ok(11, pAlice, deposit(pool, 100))
	p.ok(12, pSettler, settle(pool, 1, 11, pW1, 100))

	far := int64(10 + 525600*20)
	p.fail(far, pAlice, poolOp("refund", pool), "not closed")
	p.fail(far, pCreator, poolOp("withdraw", pool), "not closed")
	p.ok(far, pCreator, configOp(pool, pSettler2))
	p.ok(far+1, pBob, deposit(pool, 1))
}

func TestPrizeCloseWithdraw(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	reserve := p.reserve(pool)
	p.ok(11, pAlice, deposit(pool, 100))
	p.ok(12, pBob, deposit(pool, 40))
	p.tx(func(tx *gorm.DB) error {
		return p.e.dbc.TransferDrc20(tx, prizeTick, pDonor, reserve, big.NewInt(7), "donate", 12, false)
	})
	p.ok(13, pSettler, settle(pool, 1, 11, pW1, 60))

	p.fail(14, pCreator, poolOp("withdraw", pool), "not closed")
	p.fail(14, pSettler, poolOp("close", pool), "not the creator")
	p.ok(14, pCreator, poolOp("close", pool))
	p.fail(15, pCreator, poolOp("close", pool), "closed")
	p.fail(15, pSettler, poolOp("withdraw", pool), "not the creator")

	// balance 147 - owed 60 - pending 40 (Bob)
	w := p.ok(15, pCreator, poolOp("withdraw", pool))
	if p.paid(w) != 47 || p.balance(pCreator) != 1047 || p.balance(reserve) != 100 {
		t.Fatalf("withdraw paid %d, creator %d, reserve %d", p.paid(w), p.balance(pCreator), p.balance(reserve))
	}
	p.fail(16, pCreator, poolOp("withdraw", pool), "nothing to withdraw")
	p.fail(16, pAlice, deposit(pool, 1), "closed")
	p.fail(16, pSettler, settle(pool, 2, 12, pW1, 60), "closed")

	p.ok(16, pW1, poolOp("claim", pool))
	p.ok(16, pBob, poolOp("refund", pool))
	p.fail(16, pAlice, poolOp("refund", pool), "no pending deposit")
	if p.balance(reserve) != 0 || p.balance(pW1) != 60 || p.balance(pBob) != 1000 {
		t.Fatalf("reserve=%d w1=%d bob=%d", p.balance(reserve), p.balance(pW1), p.balance(pBob))
	}
}

func TestPrizeConfig(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	p.ok(11, pAlice, deposit(pool, 100))

	p.fail(12, pSettler, configOp(pool, pSettler2), "not the creator")
	p.fail(12, pCreator, configOp(pool, "DNotAnAddress"), "settler")
	p.ok(12, pCreator, configOp(pool, pSettler2))
	if got := p.pool(pool); got.Settler != pSettler2 {
		t.Fatalf("settler %s", got.Settler)
	}
	p.fail(13, pSettler, settle(pool, 1, 11, pW1, 10), "not the settler")
	p.ok(13, pSettler2, settle(pool, 1, 11, pW1, 10))
}

func TestPrizeCreateChecks(t *testing.T) {
	p := newPrizeEnv(t)
	create := func(tick, settler, timeout string) string {
		return fmt.Sprintf(`{"p":"prize-v1","op":"create","tick":%q,"settler":%q,"timeout":%q}`, tick, settler, timeout)
	}
	p.fail(10, pCreator, create("NONE", pSettler, "1440"), "not deployed")
	p.fail(10, pCreator, create(prizeTick, "DNotAnAddress", "1440"), "settler")
	p.fail(10, pCreator, create(prizeTick, pSettler, "1439"), "timeout")
	p.fail(10, pCreator, create(prizeTick, pSettler, "525601"), "timeout")
	p.ok(10, pCreator, create(prizeTick, pSettler, "525600"))
	p.ok(10, pCreator, create(prizeTick, pSettler, "1440"))
	p.fail(11, pAlice, deposit(fmt.Sprintf("%064x", 999), 1), "not found")
}

func TestPrizeConvertRejectsBadFormat(t *testing.T) {
	pool := fmt.Sprintf("%064x", 1)
	long := strings.Repeat("a", 64)
	for _, js := range []string{
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":"10","memo":"caf\u00e9"}`, pool),
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":"10","memo":"tab\there"}`, pool),
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":"10","memo":"%sa"}`, pool, long),
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":"+10"}`, pool),
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":"-10"}`, pool),
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":"1e3"}`, pool),
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":"010"}`, pool),
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":" 10"}`, pool),
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":""}`, pool),
		fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":10}`, pool),
		fmt.Sprintf(`{"op":"deposit","pool_id":"%s0","amt":"10"}`, pool),
		fmt.Sprintf(`{"op":"settle","pool_id":%q,"seq":"1","upto":"9","entries":[["%s"]]}`, pool, pW1),
		fmt.Sprintf(`{"op":"settle","pool_id":%q,"seq":"1","upto":"9","entries":[["%s","-1"]]}`, pool, pW1),
		fmt.Sprintf(`{"op":"settle","pool_id":%q,"seq":"1","upto":"9","entries":[["%s",5]]}`, pool, pW1),
		fmt.Sprintf(`{"op":"settle","pool_id":%q,"seq":"99999999999999999999","upto":"9","entries":[["%s","5"]]}`, pool, pW1),
		`{"op":"create","tick":"PRZ","settler":"x","timeout":1440}`,
		`{"op":"create","tick":"PRZ","settler":"x","timeout":"1440.0"}`,
		`{"op":"mint"}`,
		// 4-byte UTF-8 in any stored string field would fail on a utf8 (not utf8mb4) MySQL
		`{"op":"claim","pool_id":"\ud83d\udc36"}`,
		`{"op":"config","pool_id":"x","settler":"D\ud83d\udc36"}`,
		fmt.Sprintf(`{"op":"settle","pool_id":%q,"seq":"1","upto":"9","entries":[["\ud83d\udc36","5"]]}`, pool),
	} {
		param := &models.PrizeInscription{}
		if err := json.Unmarshal([]byte(js), param); err != nil {
			continue // rejected by decode's json.Unmarshal
		}
		if _, err := utils.ConvertPrize(param); err == nil {
			t.Errorf("accepted %s", js)
		}
	}

	param := &models.PrizeInscription{}
	json.Unmarshal([]byte(fmt.Sprintf(`{"op":"deposit","pool_id":%q,"amt":"0","memo":"%s"}`, pool, long)), param)
	if info, err := utils.ConvertPrize(param); err != nil || info.Memo != long {
		t.Fatalf("64-byte memo: %v", err)
	}
}

func TestPrizeDepositChecks(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	p.fail(11, pAlice, deposit(pool, 0), "amt")
	p.fail(11, pAlice, deposit(pool, 1001), "insufficient balance")
	p.fail(11, pW1, deposit(pool, 1), "transfer err") // holds no PRZ at all
	p.ok(11, pAlice, fmt.Sprintf(`{"p":"prize-v1","op":"deposit","pool_id":%q,"amt":"1","memo":"%s"}`, pool, strings.Repeat("~", 64)))
}

// a write that fails after the DRC-20 transfer rolls the transfer back too
func TestPrizeOpIsAtomic(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	fail := true
	err := p.e.dbc.DB.Callback().Create().Before("gorm:create").Register("test:fail_prize_deposit", func(db *gorm.DB) {
		if fail && db.Statement.Table == "prize_deposit" {
			db.AddError(errors.New("injected deposit write failure"))
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	p.fail(11, pAlice, deposit(pool, 100), "injected")
	if p.balance(pAlice) != 1000 || p.balance(p.reserve(pool)) != 0 {
		t.Fatalf("failed deposit moved tokens: alice=%d reserve=%d", p.balance(pAlice), p.balance(p.reserve(pool)))
	}
	var reverts int64
	p.e.dbc.DB.Model(&models.Drc20Revert{}).Where("block_number = ?", 11).Count(&reverts)
	if reverts != 0 {
		t.Fatalf("failed deposit left %d drc20_revert rows", reverts)
	}
	fail = false
	p.ok(12, pAlice, deposit(pool, 100))
}

// inscriptionVin builds the scriptSig that reDecode reads json from.
func inscriptionVin(t *testing.T, js string) btcjson.Vin {
	inner, err := txscript.NewScriptBuilder().AddData([]byte("ord")).AddData([]byte("text/plain")).AddData([]byte("xx")).AddData([]byte(js)).Script()
	if err != nil {
		t.Fatal(err)
	}
	outer, err := txscript.NewScriptBuilder().AddData([]byte("sig")).AddData([]byte("key")).AddData(inner).Script()
	if err != nil {
		t.Fatal(err)
	}
	return btcjson.Vin{Txid: commitTxid, Vout: 0, ScriptSig: &btcjson.ScriptSig{Hex: hex.EncodeToString(outer)}}
}

func TestPrizeScannerActivation(t *testing.T) {
	p := newPrizeEnv(t)
	e := p.e
	e.prizeV1Height = 100
	e.reserveSenderHeight = 1
	senderChain(t, e, pCreator)
	js := fmt.Sprintf(`{"p":"prize-v1","op":"create","tick":"PRZ","settler":%q,"timeout":"1440"}`, pSettler)
	reveal := func(txid string) *btcjson.TxRawResult {
		return &btcjson.TxRawResult{Txid: txid, Hash: txid, Vin: []btcjson.Vin{inscriptionVin(t, js)}, Vout: []btcjson.Vout{out(pCreator)}}
	}
	count := func() (n int64) {
		e.dbc.DB.Model(&models.PrizeInfo{}).Count(&n)
		return
	}

	e.currentHeight = 99
	if err := e.scanTx("r1", reveal(fmt.Sprintf("%064x", 0xa1))); err != nil || count() != 0 {
		t.Fatalf("before activation: err=%v rows=%d", err, count())
	}
	e.prizeV1Height = 0
	e.currentHeight = 1000
	if err := e.scanTx("r1", reveal(fmt.Sprintf("%064x", 0xa1))); err != nil || count() != 0 {
		t.Fatalf("disabled: err=%v rows=%d", err, count())
	}

	e.prizeV1Height = 100
	e.currentHeight = 100
	id := fmt.Sprintf("%064x", 0xa2)
	if err := e.scanTx(id, reveal(id)); err != nil {
		t.Fatal(err)
	}
	if got := p.pool(id); got.Creator != pCreator || got.BlockNumber != 100 {
		t.Fatalf("pool %+v", got)
	}

	// the holder must be the commit's sender
	id = fmt.Sprintf("%064x", 0xa3)
	other := reveal(id)
	other.Vout = []btcjson.Vout{out(pAlice)}
	if err := e.scanTx(id, other); err != nil || count() != 1 {
		t.Fatalf("holder mismatch: err=%v rows=%d", err, count())
	}

	// a deposit with a bad memo is not stored at all
	senderChain(t, e, pAlice)
	js = fmt.Sprintf(`{"p":"prize-v1","op":"deposit","pool_id":%q,"amt":"1","memo":"\u00e9"}`, fmt.Sprintf("%064x", 0xa2))
	id = fmt.Sprintf("%064x", 0xa4)
	bad := reveal(id)
	bad.Vout = []btcjson.Vout{out(pAlice)}
	if err := e.scanTx(id, bad); err != nil || count() != 1 {
		t.Fatalf("bad memo: err=%v rows=%d", err, count())
	}
}

// the pool reserve is a protocol reserve: it can never send an inscription
func TestPrizeReserveIsReserve(t *testing.T) {
	p := newPrizeEnv(t)
	pool := p.create(10, 1440)
	reserve := p.reserve(pool)
	script, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddData([]byte("prize-v1:" + pool)).Script()
	if want, _ := btcutil.NewAddressScriptHash(script, &chaincfg.MainNetParams); want.EncodeAddress() != reserve {
		t.Fatalf("reserve %s, want %s", reserve, want.EncodeAddress())
	}
	if ok, err := p.e.isReserve(reserve); err != nil || !ok {
		t.Fatalf("isReserve = %v, %v", ok, err)
	}
}

// prizeHistory is a history over heights 10..20 across two pools; want is whether
// each op succeeds.
type prizeStep struct {
	h      int64
	holder string
	js     func(p1, p2 string) string
	want   bool
}

func prizeHistory() []prizeStep {
	P1 := func(f func(string) string) func(p1, p2 string) string {
		return func(p1, _ string) string { return f(p1) }
	}
	return []prizeStep{
		{10, pCreator, func(_, _ string) string {
			return fmt.Sprintf(`{"p":"prize-v1","op":"create","tick":"PRZ","settler":%q,"timeout":"1440"}`, pSettler)
		}, true},
		{11, pAlice, P1(func(p string) string { return deposit(p, 100) }), true},
		{12, pBob, P1(func(p string) string { return deposit(p, 50) }), true},
		{13, "", nil, true}, // donation of 5 into the reserve
		{13, pSettler, P1(func(p string) string { return settle(p, 1, 12, pW1, 100, pW2, 40) }), true},
		{14, pW1, P1(func(p string) string { return poolOp("claim", p) }), true},
		{14, pAlice, P1(func(p string) string { return deposit(p, 30) }), true},
		{15, pSettler, P1(func(p string) string { return settle(p, 2, 14, pW1, 120, pW3, 10) }), true},
		{15, pW1, P1(func(p string) string { return poolOp("claim", p) }), true},
		{15, pW1, P1(func(p string) string { return poolOp("claim", p) }), false},
		{16, pW2, P1(func(p string) string { return poolOp("claim", p) }), true},
		{16, pBob, P1(func(p string) string { return deposit(p, 20) }), true},
		{17, pCreator, P1(func(p string) string { return configOp(p, pSettler2) }), true},
		{17, pCreator, func(_, _ string) string {
			return fmt.Sprintf(`{"p":"prize-v1","op":"create","tick":"PRZ","settler":%q,"timeout":"1440"}`, pSettler)
		}, true},
		{18, pAlice, func(_, p2 string) string { return deposit(p2, 10) }, true},
		{18, pSettler2, P1(func(p string) string { return settle(p, 3, 15, pW2, 50) }), true},
		{19, pCreator, P1(func(p string) string { return poolOp("close", p) }), true},
		{19, pBob, P1(func(p string) string { return poolOp("refund", p) }), true},
		{19, pAlice, P1(func(p string) string { return poolOp("refund", p) }), false},
		{20, pCreator, P1(func(p string) string { return poolOp("withdraw", p) }), true},
		{20, pW3, P1(func(p string) string { return poolOp("claim", p) }), true},
		{20, pSettler, func(_, p2 string) string { return settle(p2, 1, 18, pW1, 10) }, true},
	}
}

func stepTxid(i int) string { return fmt.Sprintf("%064x", i+1) }

// applyHistory runs the steps at heights in (from, upto]. Each step has a fixed
// txid, so a replay after a fork reuses the txids of the first pass, as a reorg does.
func applyHistory(t *testing.T, p *prizeEnv, from, upto int64) {
	steps := prizeHistory()
	var creates []string
	for i, s := range steps {
		if s.js != nil && strings.Contains(s.js("", ""), `"create"`) {
			creates = append(creates, stepTxid(i))
		}
	}
	p1, p2 := creates[0], creates[1]
	for i, s := range steps {
		if s.h <= from || s.h > upto {
			continue
		}
		if s.js == nil {
			reserve := p.reserve(p1)
			p.tx(func(tx *gorm.DB) error {
				return p.e.dbc.TransferDrc20(tx, prizeTick, pDonor, reserve, big.NewInt(5), "donate", s.h, false)
			})
			continue
		}
		p.n = i
		if _, err := p.op(s.h, s.holder, s.js(p1, p2)); (err == nil) != s.want {
			t.Fatalf("step %d h%d: want ok=%v, got %v", i, s.h, s.want, err)
		}
	}
}

// newForkEnv has every table fork() touches. Some revert models have no id, while
// their forks order by it, as the production tables have one.
func newForkEnv(t *testing.T) *prizeEnv {
	p := newPrizeEnv(t, allTables...)
	db := p.e.dbc.DB
	for _, m := range allTables {
		if db.Migrator().HasColumn(m, "id") {
			continue
		}
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			t.Fatal(err)
		}
		if err := db.Exec("ALTER TABLE `" + stmt.Schema.Table + "` ADD COLUMN id INTEGER").Error; err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func forkTo(t *testing.T, e *Explorer, height int64) {
	tx := e.dbc.DB.Begin()
	if err := e.fork(tx, height); err != nil {
		tx.Rollback()
		t.Fatalf("fork %d: %v", height, err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
}

// a fork leaves exactly the state of an explorer that only saw the blocks up to the
// fork height
func TestPrizeForkMatchesReplay(t *testing.T) {
	full := newForkEnv(t)
	applyHistory(t, full, 0, 20)

	// sanity: the history actually reached the interesting states
	if !strings.Contains(prizeState(t, full.e), "refunded=true") || !strings.Contains(prizeState(t, full.e), "status=closed") {
		t.Fatalf("history did not refund / close:\n%s", prizeState(t, full.e))
	}

	for _, h := range []int64{18, 15, 12, 10, 9} {
		forkTo(t, full.e, h)
		fresh := newForkEnv(t)
		applyHistory(t, fresh, 0, h)
		if got, want := prizeState(t, full.e), prizeState(t, fresh.e); got != want {
			t.Fatalf("fork to %d:\ngot:\n%s\nwant:\n%s", h, got, want)
		}
		if got, want := prizeInfos(t, full.e), prizeInfos(t, fresh.e); got != want {
			t.Fatalf("fork to %d infos:\ngot:\n%s\nwant:\n%s", h, got, want)
		}
	}
	var n int64
	full.e.dbc.DB.Model(&models.PrizePool{}).Count(&n)
	if n != 0 {
		t.Fatalf("pools left after fork below create: %d", n)
	}
}

// a fork straight from 20 to 15, and one from 20 to below the first create
func TestPrizeForkSingleStep(t *testing.T) {
	for _, h := range []int64{15, 9} {
		full := newForkEnv(t)
		applyHistory(t, full, 0, 20)
		forkTo(t, full.e, h)
		fresh := newForkEnv(t)
		applyHistory(t, fresh, 0, h)
		if got, want := prizeState(t, full.e), prizeState(t, fresh.e); got != want {
			t.Fatalf("fork to %d:\ngot:\n%s\nwant:\n%s", h, got, want)
		}

		// replaying the same blocks leads to the same ledger as the first pass
		applyHistory(t, full, h, 20)
		fresh = newForkEnv(t)
		applyHistory(t, fresh, 0, 20)
		if got, want := prizeState(t, full.e), prizeState(t, fresh.e); got != want {
			t.Fatalf("replay after fork to %d:\ngot:\n%s\nwant:\n%s", h, got, want)
		}
		if got, want := prizeInfos(t, full.e), prizeInfos(t, fresh.e); got != want {
			t.Fatalf("replay after fork to %d infos:\ngot:\n%s\nwant:\n%s", h, got, want)
		}
	}
}
