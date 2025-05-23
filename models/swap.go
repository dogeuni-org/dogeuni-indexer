package models

import "math/big"

type SwapInfo struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	OrderId       string    `json:"order_id"`
	Op            string    `json:"op"`
	Tick          string    `gorm:"-" json:"tick"`
	Tick0         string    `json:"tick0"`
	Tick1         string    `json:"tick1"`
	Amt0          *Number   `json:"amt0"`
	Amt1          *Number   `json:"amt1"`
	Amt0Min       *Number   `gorm:"default:'0'" json:"amt0_min"`
	Amt1Min       *Number   `gorm:"default:'0'" json:"amt1_min"`
	Amt0Out       *Number   `gorm:"default:'0'" json:"amt0_out"`
	Amt1Out       *Number   `gorm:"default:'0'" json:"amt1_out"`
	Liquidity     *Number   `json:"liquidity"`
	Doge          int       `json:"doge"`
	HolderAddress string    `json:"holder_address"`
	FeeAddress    string    `json:"fee_address"`
	FeeTxHash     string    `json:"fee_tx_hash"`
	FeeTxIndex    uint32    `gorm:"-" json:"fee_tx_index"`
	TxHash        string    `json:"tx_hash"`
	TxIndex       int       `json:"tx_index" `
	BlockNumber   int64     `json:"block_number"`
	BlockHash     string    `json:"block_hash"`
	OrderStatus   int64     `gorm:"default:1" json:"order_status"`
	ErrInfo       string    `json:"err_info"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (SwapInfo) TableName() string {
	return "swap_info"
}

type SwapLiquidity struct {
	Tick            string  `json:"tick"`
	Tick0           string  `json:"tick0"`
	Tick1           string  `json:"tick1"`
	Amt0            *Number `json:"amt0"`
	Amt1            *Number `json:"amt1"`
	LiquidityTotal  *Number `json:"liquidity_total"`
	ClosePrice      float64 `json:"close_price"`
	ReservesAddress string  `json:"reserves_address"`
	HolderAddress   string  `json:"holder_address"`
}

func (SwapLiquidity) TableName() string {
	return "swap_liquidity"
}

type SwapLiquidityLP struct {
	Tick          string   `json:"tick"`
	Liquidity     *big.Int `json:"liquidity"`
	HolderAddress string   `json:"holder_address"`
}

func (SwapLiquidityLP) TableName() string {
	return "swap_liquidity_lp"
}

type SwapRevert struct {
	ID          uint      `gorm:"primarykey"`
	Op          string    `json:"op"`
	Tick        string    `json:"tick"`
	BlockNumber int64     `json:"block_number"`
	UpdateDate  LocalTime `json:"update_date"`
	CreateDate  LocalTime `json:"create_date"`
}

func (SwapRevert) TableName() string {
	return "swap_revert"
}
