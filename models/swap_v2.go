package models

type SwapV2Info struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	OrderId       string    `json:"order_id"`
	Op            string    `json:"op"`
	PairId        string    `json:"pair_id"`
	Tick          string    `gorm:"-" json:"tick"`
	Tick0         string    `json:"tick0"`
	Tick0Id       string    `json:"tick0_id"`
	Tick1         string    `json:"tick1"`
	Tick1Id       string    `json:"tick1_id"`
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
	BlockTime     int64     `json:"block_time"`
	OrderStatus   int64     `gorm:"default:1" json:"order_status"`
	ErrInfo       string    `json:"err_info"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (SwapV2Info) TableName() string {
	return "swap_v2_info"
}

type SwapV2Liquidity struct {
	ID              uint    `gorm:"primarykey" json:"id"`
	PairId          string  `json:"pair_id"`
	Tick            string  `gorm:"-" json:"tick"`
	Tick0Id         string  `json:"tick0_id"`
	Tick1Id         string  `json:"tick1_id"`
	Tick0           string  `json:"tick0"`
	Tick1           string  `json:"tick1"`
	Amt0            *Number `json:"amt0"`
	Amt1            *Number `json:"amt1"`
	LiquidityTotal  *Number `json:"liquidity_total"`
	ClosePrice      float64 `json:"close_price"`
	ReservesAddress string  `json:"reserves_address"`
	HolderAddress   string  `json:"holder_address"`
}

func (SwapV2Liquidity) TableName() string {
	return "swap_v2_liquidity"
}

type SwapV2Revert struct {
	ID          uint      `gorm:"primarykey"`
	Op          string    `json:"op"`
	PairId      string    `json:"pair_id"`
	BlockNumber int64     `json:"block_number"`
	UpdateDate  LocalTime `json:"update_date"`
	CreateDate  LocalTime `json:"create_date"`
}

func (SwapV2Revert) TableName() string {
	return "swap_v2_revert"
}

type SwapV2Summary struct {
	ID           uint    `gorm:"primarykey" json:"id"`
	Tick         string  `json:"tick"`
	Tick0        string  `json:"tick0"`
	Tick1        string  `json:"tick1"`
	OpenPrice    float64 `json:"open_price"`
	ClosePrice   float64 `json:"close_price"`
	LowestAsk    float64 `json:"lowest_ask"`
	HighestBid   float64 `json:"highest_bid"`
	BaseVolume   *Number `json:"base_volume"`
	LastDate     string  `json:"last_date"`
	DateInterval string  `json:"date_interval"`
	DogeUsdt     float64 `json:"doge_usdt"`
}

func (SwapV2Summary) TableName() string {
	return "swap_v2_summary"
}

type SwapV2SummaryLiquidity struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	Tick         string    `json:"tick"`
	Tick0        string    `json:"tick0"`
	Tick1        string    `json:"tick1"`
	OpenPrice    float64   `json:"open_price"`
	ClosePrice   float64   `json:"close_price"`
	LowestAsk    float64   `json:"lowest_ask"`
	HighestBid   float64   `json:"highest_bid"`
	BaseVolume   *Number   `json:"base_volume"`
	QuoteVolume  *Number   `json:"quote_volume"`
	Liquidity    float64   `json:"liquidity"`
	LastDate     string    `json:"last_date"`
	DateInterval string    `json:"date_interval"`
	DogeUsdt     float64   `json:"doge_usdt"`
	UpdateDate   LocalTime `json:"update_date"`
	CreateDate   LocalTime `json:"create_date"`
}

func (SwapV2SummaryLiquidity) TableName() string {
	return "swap_v2_summary_liquidity"
}
