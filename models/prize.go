package models

// prize-v1: a protocol-held DRC-20 reserve that a settler allocates and holders claim
// from. See docs/prize-v1.md.

const (
	PrizeStatusOpen   = "open"
	PrizeStatusClosed = "closed"
)

// PrizeInfo is one prize-v1 inscription. Fields an op does not use stay zero.
type PrizeInfo struct {
	ID      uint   `gorm:"primarykey" json:"id"`
	OrderId string `json:"order_id"`
	Op      string `gorm:"size:16" json:"op"`
	PoolId  string `gorm:"size:64;index" json:"pool_id"`
	Tick    string `json:"tick"`
	Settler string `json:"settler"`
	Timeout int64  `json:"timeout"`
	// deposit: the amount deposited; claim, refund, withdraw: the amount paid out
	Amt  *Number `json:"amt"`
	Memo string  `gorm:"size:64" json:"memo"`
	Seq  int64   `json:"seq"`
	Upto int64   `json:"upto"`
	// settle: the entries as [[address, cum], ...], validated JSON
	Entries       string    `gorm:"type:text" json:"entries"`
	FeeTxHash     string    `json:"fee_tx_hash"`
	TxHash        string    `gorm:"size:64;index" json:"tx_hash"`
	BlockHash     string    `json:"block_hash"`
	BlockNumber   int64     `gorm:"index" json:"block_number"`
	HolderAddress string    `json:"holder_address"`
	ErrInfo       *string   `json:"err_info"`
	OrderStatus   int64     `json:"order_status"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (PrizeInfo) TableName() string {
	return "prize_info"
}

// PrizePool is the current state of a pool. It is derived from the successful
// prize_info rows and is recomputed from them on a fork.
type PrizePool struct {
	ID               uint    `gorm:"primarykey" json:"id"`
	PoolId           string  `gorm:"size:64;uniqueIndex" json:"pool_id"`
	Tick             string  `json:"tick"`
	Creator          string  `json:"creator"`
	Settler          string  `json:"settler"`
	Timeout          int64   `json:"timeout"`
	ReservesAddress  string  `gorm:"size:64;index" json:"reserves_address"`
	Seq              int64   `json:"seq"`
	Upto             int64   `json:"upto"`
	LastSettleHeight int64   `json:"last_settle_height"`
	Owed             *Number `json:"owed"`
	Status           string  `gorm:"size:8" json:"status"`
	// the create height
	BlockNumber int64     `json:"block_number"`
	UpdateDate  LocalTime `json:"update_date"`
	CreateDate  LocalTime `json:"create_date"`
}

func (PrizePool) TableName() string {
	return "prize_pool"
}

// PrizeEntitlement is what the settler has published for one address of a pool.
type PrizeEntitlement struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	PoolId     string    `gorm:"size:64;uniqueIndex:idx_prize_entitlement" json:"pool_id"`
	Address    string    `gorm:"size:64;uniqueIndex:idx_prize_entitlement" json:"address"`
	Cum        *Number   `json:"cum"`
	Claimed    *Number   `json:"claimed"`
	UpdateDate LocalTime `json:"update_date"`
	CreateDate LocalTime `json:"create_date"`
}

func (PrizeEntitlement) TableName() string {
	return "prize_entitlement"
}

// PrizeDeposit is one deposit. Refunded is recomputed on a fork.
type PrizeDeposit struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	PoolId        string    `gorm:"size:64;index" json:"pool_id"`
	TxHash        string    `gorm:"size:64;index" json:"tx_hash"`
	HolderAddress string    `json:"holder_address"`
	Amt           *Number   `json:"amt"`
	Memo          string    `gorm:"size:64" json:"memo"`
	BlockNumber   int64     `gorm:"index" json:"block_number"`
	Refunded      bool      `json:"refunded"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (PrizeDeposit) TableName() string {
	return "prize_deposit"
}

// PrizeSettleEntry is one entry of a successful settle.
type PrizeSettleEntry struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	PoolId      string    `gorm:"size:64;index" json:"pool_id"`
	TxHash      string    `gorm:"size:64;index" json:"tx_hash"`
	Address     string    `json:"address"`
	Cum         *Number   `json:"cum"`
	BlockNumber int64     `gorm:"index" json:"block_number"`
	UpdateDate  LocalTime `json:"update_date"`
	CreateDate  LocalTime `json:"create_date"`
}

func (PrizeSettleEntry) TableName() string {
	return "prize_settle_entry"
}

// PrizeEntry is one parsed [address, cum] pair of a settle.
type PrizeEntry struct {
	Address string
	Cum     *Number
}

// PrizeInscription is the JSON of a prize-v1 inscription. Integers are decimal
// strings; entries is [[address, cum], ...].
type PrizeInscription struct {
	P       string     `json:"p"`
	Op      string     `json:"op"`
	PoolId  string     `json:"pool_id"`
	Tick    string     `json:"tick"`
	Settler string     `json:"settler"`
	Timeout string     `json:"timeout"`
	Amt     string     `json:"amt"`
	Memo    string     `json:"memo"`
	Seq     string     `json:"seq"`
	Upto    string     `json:"upto"`
	Entries [][]string `json:"entries"`
}
