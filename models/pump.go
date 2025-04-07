package models

type PumpInfo struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	OrderId       string    `json:"order_id"`
	P             string    `json:"p"`
	Op            string    `json:"op"`
	PairId        string    `json:"pair_id"`
	Tick0         string    `json:"tick0"`
	Tick0Id       string    `json:"tick0_id"`
	Tick1         string    `json:"tick1"`
	Tick1Id       string    `json:"tick1_id"`
	Amt0          *Number   `json:"amt0"`
	Amt1          *Number   `json:"amt1"`
	Amt0Min       *Number   `json:"amt0_min"`
	Amt1Min       *Number   `json:"amt1_min"`
	Amt0Out       *Number   `gorm:"default:'0'" json:"amt0_out"`
	Amt1Out       *Number   `gorm:"default:'0'" json:"amt1_out"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	Logo          string    `json:"logo"`
	Reserve       int       `json:"reserve"`
	Doge          int       `json:"doge"`
	HolderAddress string    `json:"holder_address"`
	FeeAddress    string    `json:"fee_address"`
	FeeTxHash     string    `json:"fee_tx_hash"`
	FeeTxIndex    uint32    `gorm:"-" json:"fee_tx_index"`
	TxHash        string    `json:"tx_hash"`
	TxIndex       int       `json:"tx_index" `
	BlockNumber   int64     `json:"block_number"`
	BlockHash     string    `json:"block_hash"`
	BlockTime     int64     `json:"block_time"`
	ErrInfo       string    `json:"err_info"`
	OrderStatus   int64     `json:"order_status"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (PumpInfo) TableName() string {
	return "pump_info"
}

type PumpLiquidity struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	Tick0           string    `json:"tick0"`
	Tick0Id         string    `json:"tick0_id"`
	Tick1           string    `json:"tick1"`
	Tick1Id         string    `json:"tick1_id"`
	Amt0            *Number   `json:"amt0"`
	Amt1            *Number   `json:"amt1"`
	ReservesAddress string    `json:"reserves_address"`
	HolderAddress   string    `json:"holder_address"`
	KingDate        LocalTime `json:"king_date"`
	UpdateDate      LocalTime `json:"update_date"`
	CreateDate      LocalTime `json:"create_date"`
}

func (PumpLiquidity) TableName() string {
	return "pump_liquidity"
}

type PumpRevert struct {
	ID          uint      `gorm:"primarykey"`
	Op          string    `json:"op"`
	TickId      string    `json:"tick_id"`
	Amt0        *Number   `json:"amt0"`
	Amt1        *Number   `json:"amt1"`
	BlockNumber int64     `json:"block_number"`
	UpdateDate  LocalTime `json:"update_date"`
	CreateDate  LocalTime `json:"create_date"`
}

func (PumpRevert) TableName() string {
	return "pump_revert"
}

type PumpInviteReward struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	HolderAddress string    `json:"holder_address"`
	InviteAddress string    `json:"invite_address"`
	InviteReward  *Number   `json:"invite_reward"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (PumpInviteReward) TableName() string {
	return "pump_invite_reward"
}

type PumpInviteRewardRevert struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	HolderAddress string    `json:"holder_address"`
	InviteAddress string    `json:"invite_address"`
	InviteReward  *Number   `json:"invite_reward"`
	BlockNumber   int64     `json:"block_number"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (PumpInviteRewardRevert) TableName() string {
	return "pump_invite_reward_revert"
}
