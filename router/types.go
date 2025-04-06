package router

import "dogeuni-indexer/models"

// Pump
type PumpBoard struct {
	Tick          string           `json:"tick"`
	TickId        string           `json:"tick_id"`
	Logo          string           `json:"logo"`
	Reserve       int              `json:"reserve"`
	Tag           string           `json:"tag"`
	Twitter       *string          `json:"twitter"`
	Telegram      *string          `json:"telegram"`
	Discord       *string          `json:"discord"`
	Website       *string          `json:"website"`
	Youtube       *string          `json:"youtube"`
	Tiktok        *string          `json:"tiktok"`
	Name          string           `json:"name"`
	Holders       int              `json:"holders"`
	Description   string           `json:"description"`
	Transactions  int              `json:"transactions"`
	Replies       int              `json:"replies"`
	Amt0          *models.Number   `json:"amt0"`
	Amt1          *models.Number   `json:"amt1"`
	HolderAddress string           `json:"holder_address"`
	ProfilePhoto  string           `json:"profile_photo"`
	UserName      string           `json:"user_name"`
	Bio           string           `json:"bio"`
	KingDate      models.LocalTime `json:"king_date"`
	CreateDate    models.LocalTime `json:"create_date"`
}

type MergeSwapPumpOrderResult struct {
	P             string           `json:"p"`
	Op            string           `json:"op"`
	Tick0ID       string           `json:"tick0_id"`
	Tick0         string           `json:"tick0"`
	Tick1ID       string           `json:"tick1_id"`
	Tick1         string           `json:"tick1"`
	Amt0          *models.Number   `json:"amt0"`
	Amt1          *models.Number   `json:"amt1"`
	Amt0Out       *models.Number   `gorm:"default:'0'" json:"amt0_out"`
	Amt1Out       *models.Number   `gorm:"default:'0'" json:"amt1_out"`
	TxHash        string           `json:"tx_hash"`
	HolderAddress string           `json:"holder_address"`
	OrderStatus   int              `json:"order_status"`
	CreateDate    models.LocalTime `json:"create_date"`
}
