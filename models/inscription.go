package models

type BaseInscription struct {
	P  string `json:"p"`
	Op string `json:"op"`
}

type Drc20Inscription struct {
	P    string `json:"p"`
	Op   string `json:"op"`
	Tick string `json:"tick"`
	Max  string `json:"max"`
	Amt  string `json:"amt"`
	Lim  string `json:"lim"`
	Dec  uint   `json:"dec"`
	Burn string `json:"burn"`
	Func string `json:"func"`
}

type SwapInscription struct {
	P         string `json:"p"`
	Op        string `json:"op"`
	Tick0     string `json:"tick0"`
	Tick1     string `json:"tick1"`
	Amt0      string `json:"amt0"`
	Amt1      string `json:"amt1"`
	Amt0Min   string `json:"amt0_min"`
	Amt1Min   string `json:"amt1_min"`
	Liquidity string `json:"liquidity"`
	Doge      int    `json:"doge"`
}

type WDogeInscription struct {
	P             string `json:"p"`
	Op            string `json:"op"`
	Tick          string `json:"tick"`
	Amt           string `json:"amt"`
	HolderAddress string `json:"holder_address"`
}

type CrossInscription struct {
	P             string `json:"p"`
	Op            string `json:"op"`
	Chain         string `json:"chain"`
	Tick          string `json:"tick"`
	Amt           string `json:"amt"`
	AdminAddress  string `json:"admin_address"`
	ToAddress     string `json:"to_address"`
	HolderAddress string `json:"holder_address"`
}

type NftInscription struct {
	P      string `json:"p"`
	Op     string `json:"op"`
	Tick   string `json:"tick"`
	TickId int64  `json:"tick_id"`
	Total  int64  `json:"total"`
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Seed   int64  `json:"seed"`
	Image  string `json:"image"`
}

type FileInscription struct {
	P      string `json:"p"`
	Op     string `json:"op"`
	Tick   string `json:"tick"`
	FileId string `json:"file_id"`
	File   []byte `json:"-"`
}

type StakeInscription struct {
	P    string `json:"p"`
	Op   string `json:"op"`
	Tick string `json:"tick"`
	Amt  string `json:"amt"`
}

type StakeV2Inscription struct {
	P          string `json:"p"`
	Op         string `json:"op"`
	StakeId    string `json:"stake_id"`
	Tick0      string `json:"tick0"`
	Tick1      string `json:"tick1"`
	Amt        string `json:"amt"`
	Reward     string `json:"reward"`
	EachReward string `json:"each_reward"`
	LockBlock  int64  `json:"lock_block"`
}

type ExchangeInscription struct {
	P     string `json:"p"`
	Op    string `json:"op"`
	ExId  string `json:"exid"`
	Tick0 string `json:"tick0"`
	Tick1 string `json:"tick1"`
	Amt0  string `json:"amt0"`
	Amt1  string `json:"amt1"`
}

type FileExchangeInscription struct {
	P      string `json:"p"`
	Op     string `json:"op"`
	FileId string `json:"file_id"`
	ExId   string `json:"ex_id"`
	Tick   string `json:"tick"`
	Amt    string `json:"amt"`
}

type BoxInscription struct {
	P        string `json:"p"`
	Op       string `json:"op"`
	Tick0    string `json:"tick0"`
	Tick1    string `json:"tick1"`
	Max      string `json:"max"`
	Amt0     string `json:"amt0"`
	LiqAmt   string `json:"liqamt"`
	LiqBlock int64  `json:"liqblock"`
	Amt1     string `json:"amt1"`
}

type Meme20Inscription struct {
	P      string `json:"p"`
	Op     string `json:"op"`
	Tick   string `json:"tick"`
	TickId string `json:"tick_id"`
	Name   string `json:"name"`
	Max    string `json:"max"`
	Amt    string `json:"amt"`
}

type PumpInscription struct {
	P           string `json:"p"`
	Op          string `json:"op"`
	PairId      string `json:"pair_id"`
	Symbol      string `json:"symbol"`
	Tick        string `json:"tick"`
	Tick0Id     string `json:"tick0_id"`
	Tick1Id     string `json:"tick1_id"`
	Amt         string `json:"amt"`
	Amt0        string `json:"amt0"`
	Amt1Min     string `json:"amt1_min"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Twitter     string `json:"twitter"`
	Telegram    string `json:"telegram"`
	Website     string `json:"website"`
	Discord     string `json:"discord"`
	Logo        string `json:"logo"`
	Reserve     int    `json:"reserve"`
	Doge        int    `json:"doge"`
}

type SwapV2Inscription struct {
	P         string `json:"p"`
	Op        string `json:"op"`
	PairId    string `json:"pair_id"`
	Tick0Id   string `json:"tick0_id"`
	Tick1Id   string `json:"tick1_id"`
	Amt0      string `json:"amt0"`
	Amt1      string `json:"amt1"`
	Amt0Min   string `json:"amt0_min"`
	Amt1Min   string `json:"amt1_min"`
	Liquidity string `json:"liquidity"`
	Doge      int    `json:"doge"`
}

type InviteInscription struct {
	P             string `json:"p"`
	Op            string `json:"op"`
	InviteAddress string `json:"invite_address"`
}
