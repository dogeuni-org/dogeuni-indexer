package models

type Meme20Info struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	OrderId       string    `json:"order_id"`
	P             string    `json:"p"`
	Op            string    `json:"op"`
	TickId        string    `json:"tick_id"`
	Tick          string    `json:"tick"`
	Name          string    `json:"name"`
	Amt           *Number   `json:"amt"`
	Max           *Number   `gorm:"column:max_" json:"max"`
	Dec           uint      `gorm:"column:dec_" json:"dec"`
	HolderAddress string    `json:"holder_address"`
	ToAddress     string    `json:"to_address"`
	FeeAddress    string    `json:"fee_address"`
	FeeTxHash     string    `json:"fee_tx_hash"`
	TxHash        string    `json:"tx_hash"`
	BlockNumber   int64     `json:"block_number"`
	BlockHash     string    `json:"block_hash"`
	ErrInfo       string    `json:"err_info"`
	OrderStatus   int64     `json:"order_status"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (Meme20Info) TableName() string {
	return "meme20_info"
}

type Meme20Collect struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	TickId        string    `json:"tick_id"`
	Tick          string    `json:"tick"`
	Name          string    `json:"name"`
	Max           *Number   `gorm:"column:max_" json:"max"`
	Dec           uint      `gorm:"column:dec_" json:"dec"`
	Reserve       int       `json:"reserve"`
	HolderAddress string    `json:"holder_address"`
	Transactions  uint64    `json:"transactions"`
	Holders       uint64    `gorm:"->"  json:"holders"`
	Logo          string    `json:"logo"`
	Tag           *string   `json:"tag"`
	Description   *string   `json:"description"`
	Twitter       *string   `json:"twitter"`
	Telegram      *string   `json:"telegram"`
	Discord       *string   `json:"discord"`
	Website       *string   `json:"website"`
	Youtube       *string   `json:"youtube"`
	Tiktok        *string   `json:"tiktok"`
	IsCheck       uint64    `json:"is_check"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (Meme20Collect) TableName() string {
	return "meme20_collect"
}

type Meme20CollectAddress struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	TickId        string    `json:"tick_id"`
	Tick          string    `gorm:"->" json:"tick"`
	Name          string    `gorm:"->" json:"name"`
	Max           *Number   `gorm:"column:max_; ->" json:"max"`
	Logo          string    `gorm:"column:logo; ->" json:"logo"`
	LpAmt0        *Number   `gorm:"column:lp_amt0; ->" json:"lp_amt0"`
	LpAmt1        *Number   `gorm:"column:lp_amt1; ->" json:"lp_amt1"`
	Amt           *Number   `json:"amt"`
	HolderAddress string    `json:"holder_address"`
	Transactions  uint64    `json:"transactions"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (Meme20CollectAddress) TableName() string {
	return "meme20_collect_address"
}

type Meme20Revert struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	TickId      string    `json:"tick_id"`
	Tick        string    `gorm:"column:tick; ->" json:"tick"`
	Name        string    `gorm:"column:name; ->" json:"name"`
	Amt         *Number   `json:"amt"`
	TxHash      string    `json:"tx_hash"`
	BlockNumber int64     `json:"block_number"`
	UpdateDate  LocalTime `json:"update_date"`
	CreateDate  LocalTime `json:"create_date"`
}

func (Meme20Revert) TableName() string {
	return "meme20_revert"
}
