package models

type Summary struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	TickId       string    `json:"tick_id"`
	OpenPrice    float64   `json:"open_price"`
	ClosePrice   float64   `json:"close_price"`
	LowestAsk    float64   `json:"lowest_ask"`
	HighestBid   float64   `json:"highest_bid"`
	BaseVolume   *Number   `json:"base_volume"`
	TimeStamp    int64     `json:"time_stamp"`
	LastDate     string    `json:"last_date"`
	DateInterval string    `json:"date_interval"`
	DogeUsdt     float64   `json:"doge_usdt"`
	UpdateDate   LocalTime `json:"update_date"`
	CreateDate   LocalTime `json:"create_date"`
}

type SwapSummary struct {
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

func (SwapSummary) TableName() string {
	return "swap_summary"
}

type SwapSummaryLiquidity struct {
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

func (SwapSummaryLiquidity) TableName() string {
	return "swap_summary_liquidity"
}
