package models

import "math/big"

type ConsensusInfo struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	OrderId       string    `json:"order_id"`
	P             string    `json:"p"`              
	Op            string    `json:"op"`             
    StakeId       string    `json:"stake_id"`       
	Amt           *Number   `json:"amt"`            // 数量
	HolderAddress string    `json:"holder_address"` // 操作地址
	FeeAddress    string    `json:"fee_address"`    // 手续费地址
	FeeTxHash     string    `json:"fee_tx_hash"`    // 手续费交易哈希
	TxHash        string    `json:"tx_hash"`
	BlockNumber   int64     `json:"block_number"`
	BlockHash     string    `json:"block_hash"`
	ErrInfo       string    `json:"err_info"` // 错误信息
	OrderStatus   int64     `json:"order_status"`
	UpdateDate    LocalTime `json:"update_date"`
	CreateDate    LocalTime `json:"create_date"`
}

func (ConsensusInfo) TableName() string {
	return "consensus_info"
}


// ConsensusRevert rollback information
type ConsensusRevert struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	Op          string    `json:"op"` // stake/unstake
	Amt         *Number   `json:"amt"`
	TxHash      string    `json:"tx_hash"`
	BlockNumber int64     `json:"block_number"`
	UpdateDate  LocalTime `json:"update_date"`
	CreateDate  LocalTime `json:"create_date"`
}

func (ConsensusRevert) TableName() string {
	return "consensus_revert"
}

// ConsensusStakeRecord independent stake record (one record per stake, no stacking; only supports one-time full unstake)
type ConsensusStakeRecord struct {
    ID              uint      `gorm:"primarykey" json:"id"`
    StakeId         string    `gorm:"uniqueIndex" json:"stake_id"`  // Recommended to use stake transaction TxHash
    HolderAddress   string    `json:"holder_address"`
    ReservesAddress string    `json:"reserves_address"`
    Amt             *Number   `json:"amt"`
    StakeBlock      int64     `json:"stake_block"`
    UnstakeBlock    *int64    `json:"unstake_block"`
    Status          string    `json:"status"` // active/closed
    Score           *Number   `json:"score"`  // Score settled by block count when unstaking
    UpdateDate      LocalTime `json:"update_date"`
    CreateDate      LocalTime `json:"create_date"`
}

func (ConsensusStakeRecord) TableName() string {
    return "consensus_stake_record"
}

// ConsensusCollectAll summary information compatible with old interface
type ConsensusCollectAll struct {
	TotalStaked      *big.Int `json:"total_staked"`       // Total staked amount
	TotalUnstaked    *big.Int `json:"total_unstaked"`     // Total unstaked amount
	NetStaked        *big.Int `json:"net_staked"`         // Net staked amount
	Holders          uint64   `json:"holders"`            // Number of stakers
	Transactions     uint64   `json:"transactions"`       // Number of transactions
	LastStakeBlock   *int64   `json:"last_stake_block"`   // Last stake block
	LastUnstakeBlock *int64   `json:"last_unstake_block"` // Last unstake block
}

// ConsensusCollectAllCache cache structure
type ConsensusCollectAllCache struct {
	Results     []*ConsensusCollectAll
	Total       int64
	CacheNumber int64
}

// ConsensusCollectRouter routing structure
type ConsensusCollectRouter struct {
	TotalStaked      string `json:"total_staked"`       // Total staked amount
	TotalUnstaked    string `json:"total_unstaked"`     // Total unstaked amount
	NetStaked        string `json:"net_staked"`         // Net staked amount
	Holders          uint64 `json:"holders"`            // Number of stakers
	Transactions     uint64 `json:"transactions"`       // Number of transactions
	LastStakeBlock   *int64 `json:"last_stake_block"`   // Last stake block
	LastUnstakeBlock *int64 `json:"last_unstake_block"` // Last unstake block
}

// ConsensusCollectCache cache structure
type ConsensusCollectCache struct {
	Results     []*ConsensusCollectRouter
	Total       int64
	CacheNumber int64
}

// ConsensusCreditScoreRouter credit score routing structure
type ConsensusCreditScoreRouter struct {
	HolderAddress   string  `json:"holder_address"`    // Address
	CreditScore     float64 `json:"credit_score"`      // Credit score
	AWeighted       string  `json:"a_weighted"`        // Weighted amount
	TScore          float64 `json:"t_score"`           // Time factor
	EFactor         float64 `json:"e_factor"`          // Early bonus factor
	FirstStakeBlock int64   `json:"first_stake_block"` // First stake block
	LastStakeBlock  int64   `json:"last_stake_block"`  // Last stake block
	Rank            int64   `json:"rank"`              // Score ranking
	LastBlock       int64   `json:"last_block"`        // Last update block
	StakedAmt       string  `json:"staked_amt"`        // Staked amount
	UnstakedAmt     string  `json:"unstaked_amt"`      // Unstaked amount
	NetStaked       string  `json:"net_staked"`        // Net staked amount
	Transactions    uint64  `json:"transactions"`      // Number of transactions
}

// ConsensusCreditScoreCache credit score cache structure
type ConsensusCreditScoreCache struct {
	Results     []*ConsensusCreditScoreRouter
	Total       int64
	CacheNumber int64
}
