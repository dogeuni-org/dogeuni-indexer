package models

type InviteInfo struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	OrderId       string    `json:"order_id"`
	P             string    `json:"p"`
	Op            string    `json:"op"`
	HolderAddress string    `json:"holder_address"`
	InviteAddress string    `json:"invite_address"`
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

func (InviteInfo) TableName() string {
	return "invite_info"
}

type InviteCollect struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	HolderAddress string    `json:"holder_address"`
	InviteAddress string    `json:"invite_address"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (InviteCollect) TableName() string {
	return "invite_collect"
}

type InviteRevert struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	HolderAddress string    `json:"holder_address"`
	InviteAddress string    `json:"invite_address"`
	BlockNumber   int64     `json:"block_number"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}
