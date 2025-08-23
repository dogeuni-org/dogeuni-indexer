package explorer

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"dogeuni-indexer/utils"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"math/big"
	"strings"
)

var (
	Number0            = big.NewInt(0)
	maxAllowedValue, _ = big.NewInt(0).SetString("99999999999999999999999999999999999999999", 10)
	DogeMax            = big.NewInt(3300000000000)
)

const (
	MEMETICKID_LENGTH = 64
)

type Verifys struct {
	dbc *storage.DBClient
}

func NewVerifys(dbc *storage.DBClient) *Verifys {
	return &Verifys{
		dbc: dbc,
	}
}

func (v *Verifys) VerifyConsensus(c *models.ConsensusInfo) error {
	switch c.Op {
	case "stake":
		return v.verifyConsensusStake(c)
	case "unstake":
		return v.verifyConsensusUnstake(c)
	default:
		return fmt.Errorf("do not support the type of consensus op")
	}
}

func (v *Verifys) verifyConsensusStake(c *models.ConsensusInfo) error {
	if c.HolderAddress == "" {
		return fmt.Errorf("consensus verify: holder address required")
	}
	if c.StakeId == "" {
		c.StakeId = c.TxHash
	}
	// Pre-stake validation: check if CARDI balance is sufficient
	holder := &models.Drc20CollectAddress{}
	if err := v.dbc.DB.Where("tick = ? and holder_address = ?", "CARDI", c.HolderAddress).First(holder).Error; err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}
	if c.Amt.Int().Cmp(holder.AmtSum.Int()) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}
	return nil
}

func (v *Verifys) verifyConsensusUnstake(c *models.ConsensusInfo) error {
	if c.StakeId == "" {
		return fmt.Errorf("consensus verify: stake_id required")
	}
	rec := &models.ConsensusStakeRecord{}
	if err := v.dbc.DB.Where("stake_id = ? AND status = ?", c.StakeId, "active").First(rec).Error; err != nil {
		return fmt.Errorf("consensus verify: %v", err)
	}
	if rec.HolderAddress != c.HolderAddress {
		return fmt.Errorf("consensus verify: holder address mismatch")
	}
	return nil
}

func (v *Verifys) VerifyDrc20(card *models.Drc20Info) error {
	switch card.Op {
	case "deploy":
		return v.verifyDeploy(card)
	case "mint":
		return v.verifyMint(card)
	case "transfer":
		return v.verifyTransfer(card)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifyDeploy(card *models.Drc20Info) error {

	if len(card.Tick) < 2 || len(card.Tick) > 8 {
		return fmt.Errorf("the token symbol must be 2 or 8 letters")
	}

	if card.Max.Int().Cmp(Number0) < 1 || card.Lim.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if card.Max.Int().Cmp(maxAllowedValue) > 0 || card.Lim.Int().Cmp(maxAllowedValue) > 0 {
		return fmt.Errorf("the maximum value cannot be greater 99999999999999999999999999999999999999999")
	}

	if card.Max.Cmp(card.Lim) < 0 {
		return fmt.Errorf("the maximum value is less than the limit value")
	}

	err := v.dbc.DB.Where("tick = ?", card.Tick).First(&models.Drc20Collect{}).Error
	if err == nil {
		return fmt.Errorf("has been deployed contracts")
	} else {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}
	}

	return nil
}

func (v *Verifys) verifyMint(card *models.Drc20Info) error {

	if len(card.Tick) < 2 || len(card.Tick) > 8 {
		return fmt.Errorf("the token symbol must be 2 or 8 letters")
	}

	card1 := &models.Drc20Collect{}
	err := v.dbc.DB.Where("tick = ?", card.Tick).First(card1).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist")
	}

	if card.Amt.Int().Cmp(big.NewInt(0)) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if card.Amt.Cmp(card1.Lim) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the limit")
	}

	amount := big.NewInt(0).Mul(card.Amt.Int(), big.NewInt(int64(card.Repeat)))
	Amt := new(big.Int).Add(card1.AmtSum.Int(), amount)
	if Amt.Cmp(card1.Max.Int()) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the maximum")
	}

	return nil
}

func (v *Verifys) verifyTransfer(card *models.Drc20Info) error {

	if card.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	tranCount := len(strings.Split(card.ToAddress, ","))

	card1 := &models.Drc20CollectAddress{}
	err := v.dbc.DB.Where("tick = ? and holder_address = ?", card.Tick, card.HolderAddress).First(card1).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist")
	}

	CAmt := big.NewInt(0).Mul(card.Amt.Int(), big.NewInt(int64(tranCount)))
	if CAmt.Cmp(card1.AmtSum.Int()) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}
	return nil
}

func (v *Verifys) VerifySwap(tx *gorm.DB, swap *models.SwapInfo) error {
	switch swap.Op {
	case "create":
		return v.verifySwapCreate(tx, swap)
	case "add":
		return v.verifySwapAdd(tx, swap)
	case "remove":
		return v.verifySwapRemove(tx, swap)
	case "swap":
		return v.verifySwapExec(tx, swap)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifySwapCreate(tx *gorm.DB, swap *models.SwapInfo) error {

	if swap.Amt0.Int().Cmp(Number0) < 1 || swap.Amt1.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if swap.Tick0 == swap.Tick1 {
		return fmt.Errorf("the token symbol must be different")
	}

	tick0, tick1, amt0, amt1, _, _ := utils.SortTokens(swap.Tick0, swap.Tick1, swap.Amt0, swap.Amt1, nil, nil)

	err := tx.Where("tick0 = ? and tick1 = ?", tick0, tick1).First(&models.SwapLiquidity{}).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}
	} else {
		return fmt.Errorf("the contract has been created")
	}

	cardA0 := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick0, swap.HolderAddress).First(cardA0).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	cardA1 := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick1, swap.HolderAddress).First(cardA1).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if amt0.Cmp(cardA0.AmtSum) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	if amt1.Cmp(cardA1.AmtSum) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) verifySwapAdd(tx *gorm.DB, swap *models.SwapInfo) error {

	tick0, tick1, amt0, amt1, amt0Min, amt1Min := utils.SortTokens(swap.Tick0, swap.Tick1, swap.Amt0, swap.Amt1, swap.Amt0Min, swap.Amt1Min)

	if swap.Amt0.Int().Cmp(Number0) < 1 || swap.Amt1.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if swap.Tick0 == swap.Tick1 {
		return fmt.Errorf("the token symbol must be different")
	}

	swapLiquidity := &models.SwapLiquidity{}
	err := tx.Where("tick0 = ? and tick1 = ?", tick0, tick1).First(swapLiquidity).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	cardA0 := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick0, swap.HolderAddress).First(cardA0).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	cardA1 := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick1, swap.HolderAddress).First(cardA1).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	sum0 := cardA0.AmtSum
	sum1 := cardA1.AmtSum

	amountBOptimal := big.NewInt(0).Mul(amt0.Int(), swapLiquidity.Amt1.Int())

	if amountBOptimal.Cmp(big.NewInt(0)) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	amountBOptimal = big.NewInt(0).Div(amountBOptimal, swapLiquidity.Amt0.Int())

	if amountBOptimal.Cmp(amt1Min.Int()) < 0 {
		amountAOptimal := big.NewInt(0).Mul(amt1.Int(), swapLiquidity.Amt0.Int())
		if amountAOptimal.Cmp(big.NewInt(0)) < 1 {
			return fmt.Errorf("the amount of tokens exceeds the 0")
		}
		amountAOptimal = big.NewInt(0).Div(amountAOptimal, swapLiquidity.Amt1.Int())

		if amountAOptimal.Cmp(amt0Min.Int()) < 0 {
			return fmt.Errorf("the amount of tokens exceeds the min")
		} else {
			if amountAOptimal.Cmp(sum0.Int()) > 0 {
				return fmt.Errorf("the amount of tokens exceeds the balance")
			}

			if amt1.Cmp(sum1) > 0 {
				return fmt.Errorf("the amount of tokens exceeds the balance")
			}
		}
	} else {
		if amt0.Cmp(sum0) > 0 {
			return fmt.Errorf("the amount of tokens exceeds the balance")
		}

		if amountBOptimal.Cmp(sum1.Int()) > 0 {
			return fmt.Errorf("the amount of tokens exceeds the max")
		}
	}

	return nil
}

func (v *Verifys) verifySwapExec(tx *gorm.DB, swap *models.SwapInfo) error {

	if swap.Amt0.Int().Cmp(Number0) < 1 || swap.Amt1Min.Int().Cmp(Number0) < 0 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if swap.Tick0 == swap.Tick1 {
		return fmt.Errorf("the token symbol must be different")
	}

	tick0, tick1, _, _, _, _ := utils.SortTokens(swap.Tick0, swap.Tick1, swap.Amt0, swap.Amt1, swap.Amt0Min, swap.Amt1Min)

	swapLiquidity := &models.SwapLiquidity{}
	err := tx.Where("tick0 = ? and tick1 = ?", tick0, tick1).First(swapLiquidity).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	amtMap := make(map[string]*big.Int)
	amtMap[swapLiquidity.Tick0] = swapLiquidity.Amt0.Int()
	amtMap[swapLiquidity.Tick1] = swapLiquidity.Amt1.Int()

	amtfee0 := new(big.Int).Div(swap.Amt0.Int(), big.NewInt(1000))
	amtin := new(big.Int).Mul(amtfee0, big.NewInt(3))
	amtin = new(big.Int).Sub(swap.Amt0.Int(), amtin)

	amtout := new(big.Int).Mul(amtin, amtMap[swap.Tick1])
	amtout = new(big.Int).Div(amtout, new(big.Int).Add(amtMap[swap.Tick0], amtin))

	if amtout.Cmp(swap.Amt1Min.Int()) < 0 {
		return fmt.Errorf("the minimum output less than the limit.")
	}

	cardA0 := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", swap.Tick0, swap.HolderAddress).First(cardA0).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	cardA1 := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", swap.Tick1, swapLiquidity.ReservesAddress).First(cardA1).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	sum0 := cardA0.AmtSum
	sum1 := cardA1.AmtSum

	if swap.Amt0.Cmp(sum0) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance of the input token")
	}

	if amtout.Cmp(sum1.Int()) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) verifySwapRemove(tx *gorm.DB, swap *models.SwapInfo) error {

	if swap.Liquidity.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if swap.Tick0 == swap.Tick1 {
		return fmt.Errorf("the token symbol must be different")
	}

	tick0, tick1, _, _, _, _ := utils.SortTokens(swap.Tick0, swap.Tick1, swap.Amt0, swap.Amt1, swap.Amt0Min, swap.Amt1Min)

	swapLiquidity := &models.SwapLiquidity{}
	err := tx.Where("tick0 = ? and tick1 = ?", tick0, tick1).First(swapLiquidity).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if swap.Liquidity.Int().Cmp(swapLiquidity.LiquidityTotal.Int()) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	tick := tick0 + "-SWAP-" + tick1

	cardA0 := &models.Drc20CollectAddress{}
	err = tx.Where("tick = ? and holder_address = ?", tick, swap.HolderAddress).First(cardA0).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	sum0 := cardA0.AmtSum

	if swap.Liquidity.Cmp(sum0) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) VerifyWDoge(wdoge *models.WDogeInfo) error {
	switch wdoge.Op {
	case "deposit":
		return v.verifyWDogeDeposit(wdoge)
	case "withdraw":
		return v.verifyWDogeWithdraw(wdoge)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifyWDogeDeposit(wdoge *models.WDogeInfo) error {
	if wdoge.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}
	return nil
}

func (v *Verifys) verifyWDogeWithdraw(wdoge *models.WDogeInfo) error {

	if wdoge.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if wdoge.Amt.Int().Cmp(big.NewInt(100000000)) < 0 {
		return fmt.Errorf("the amount of tokens exceeds the 1")
	}

	holder := &models.Drc20CollectAddress{}
	err := v.dbc.DB.Where("tick = ? and holder_address = ?", "WDOGE(WRAPPED-DOGE)", wdoge.HolderAddress).First(holder).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	sum0 := holder.AmtSum

	if wdoge.Amt.Cmp(sum0) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) VerifyNFT(nft *models.NftInfo) error {
	switch nft.Op {
	case "deploy":
		return v.verifyNFTDeploy(nft)
	case "mint":
		return v.verifyNFTMint(nft)
	case "transfer":
		return v.verifyNFTTransfer(nft)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifyNFTDeploy(nft *models.NftInfo) error {

	if len(nft.Tick) < 2 || len(nft.Tick) > 32 {
		return fmt.Errorf("the token symbol must be 2 or 32 letters")
	}

	if nft.Total < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	nftc := &models.NftCollect{}
	err := v.dbc.DB.Where("tick = ?", nft.Tick).First(nftc).Error
	if err == nil {
		return fmt.Errorf("has been deployed contracts")
	}

	cardA0 := &models.Drc20CollectAddress{}
	err = v.dbc.DB.Where("tick = ? and holder_address = ?", "CARDI", nft.HolderAddress).First(cardA0).Error
	if err != nil {
		return errors.New("Deploying AI/NFT requires holding 8400 CARDI for deployment. Please note that holding is only for identity verification and will not affect your assets.")
	}

	sum := cardA0.AmtSum.Int()
	cardiTotal := big.NewInt(840000000000)
	if sum.Cmp(cardiTotal) < 0 {
		return fmt.Errorf("Deploying AI/NFT requires holding 8400 CARDI for deployment. Please note that holding is only for identity verification and will not affect your assets.")
	}

	return nil
}

func (v *Verifys) verifyNFTMint(nft *models.NftInfo) error {

	if len(nft.Tick) < 2 || len(nft.Tick) > 32 {
		return fmt.Errorf("the token symbol must be 2 or 32 letters")
	}

	nftc := &models.NftCollect{}
	err := v.dbc.DB.Where("tick = ?", nft.Tick).First(nftc).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist")
	}

	if nftc.Total <= nftc.TickSum {
		return fmt.Errorf("the amount of tokens exceeds the maximum")
	}

	return nil
}

func (v *Verifys) verifyNFTTransfer(nft *models.NftInfo) error {

	if nft.TickId < 0 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	nca := &models.NftCollectAddress{}
	err := v.dbc.DB.Where("tick = ? and holder_address = ? and tick_id = ?", nft.Tick, nft.HolderAddress, nft.TickId).First(nca).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	return nil
}

func (v *Verifys) VerifyFile(file *models.FileInfo) error {
	switch file.Op {
	case "deploy":
		return v.verifyFileDeploy(file)
	case "transfer":
		return v.verifyFileTransfer(file)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifyFileDeploy(file *models.FileInfo) error {
	return nil
}

func (v *Verifys) verifyFileTransfer(file *models.FileInfo) error {

	fca := &models.FileCollectAddress{}
	err := v.dbc.DB.Where("file_id = ? and holder_address = ? ", file.FileId, file.HolderAddress).First(fca).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	return nil
}

func (v *Verifys) VerifyStake(stake *models.StakeInfo) error {
	switch stake.Op {
	case "stake":
		return v.verifyStakeStake(stake)
	case "unstake":
		return v.verifyStakeUnStake(stake)
	case "getallreward":
		return v.verifyStakeGetAllReward(stake)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifyStakeStake(si *models.StakeInfo) error {

	if si.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	cardA0 := &models.Drc20CollectAddress{}
	err := v.dbc.DB.Where("tick = ? and holder_address = ?", si.Tick, si.HolderAddress).First(cardA0).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	sum0 := cardA0.AmtSum

	if si.Amt.Cmp(sum0) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) verifyStakeUnStake(si *models.StakeInfo) error {

	if si.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	sca := &models.StakeCollectAddress{}
	err := v.dbc.DB.Where("holder_address = ? and tick = ? ", si.HolderAddress, si.Tick).First(sca).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if si.Amt.Cmp(sca.Amt) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) verifyStakeGetAllReward(si *models.StakeInfo) error {

	sca := &models.StakeCollectAddress{}
	err := v.dbc.DB.Where("holder_address = ? and tick = ? ", si.HolderAddress, si.Tick).First(sca).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	return nil
}

func (v *Verifys) VerifyStakeV2(stake *models.StakeV2Info) error {
	switch stake.Op {
	case "create":
		return v.verifyStakeV2Create(stake)
	case "stake":
		return v.verifyStakeV2Stake(stake)
	case "unstake":
		return v.verifyStakeV2UnStake(stake)
	case "getreward":
		return v.verifyStakeV2GetReward(stake)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifyStakeV2Create(si *models.StakeV2Info) error {

	if si.Reward.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if si.EachReward.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if si.Reward.Int().Cmp(si.EachReward.Int()) < 0 {
		return fmt.Errorf("each reward exceeds the total reward")
	}

	err := v.dbc.DB.Where("tick = ? ", si.Tick0).First(&models.Drc20Collect{}).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s tick: %s", err.Error(), si.Tick0)
	}

	err = v.dbc.DB.Where("tick = ? ", si.Tick1).First(&models.Drc20Collect{}).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s tick: %s", err.Error(), si.Tick1)
	}

	cardA1 := &models.Drc20CollectAddress{}
	err = v.dbc.DB.Where("tick = ? and holder_address = ?", si.Tick1, si.HolderAddress).First(cardA1).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s tick: %s holder_address: %s", err.Error(), si.Tick1, si.HolderAddress)
	}

	sum0 := cardA1.AmtSum

	if si.Amt.Cmp(sum0) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) verifyStakeV2Stake(si *models.StakeV2Info) error {

	if si.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	sc := &models.StakeV2Collect{}
	err := v.dbc.DB.Where("stake_id = ? ", si.StakeId).First(sc).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if sc.LastRewardBlock != 0 {
		if si.BlockNumber > sc.LastRewardBlock {
			return fmt.Errorf("the amount of tokens exceeds the end block")
		}
	}

	cardA1 := &models.Drc20CollectAddress{}
	err = v.dbc.DB.Where("tick = ? and holder_address = ?", si.Tick1, si.HolderAddress).First(cardA1).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	return nil
}

func (v *Verifys) verifyStakeV2UnStake(si *models.StakeV2Info) error {

	if si.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	sca := &models.StakeV2CollectAddress{}
	err := v.dbc.DB.Where("holder_address = ? and stake_id = ? ", si.HolderAddress, si.StakeId).First(sca).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if sca.LastBlock > si.BlockNumber {
		return fmt.Errorf("the amount of tokens exceeds the lock block")
	}

	if si.Amt.Cmp(sca.Amt) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) verifyStakeV2GetReward(si *models.StakeV2Info) error {

	sca := &models.StakeV2CollectAddress{}
	err := v.dbc.DB.Where("holder_address = ? and stake_id = ? ", si.HolderAddress, si.StakeId).First(sca).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	return nil
}

func (v *Verifys) VerifyExchange(ex *models.ExchangeInfo) error {

	switch ex.Op {
	case "create":
		return v.VerifyExchangeCreate(ex)
	case "trade":
		return v.VerifyExchangeTrade(ex)
	case "cancel":
		return v.VerifyExchangeCancel(ex)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) VerifyExchangeCreate(ex *models.ExchangeInfo) error {

	card0 := &models.Drc20Collect{}
	err := v.dbc.DB.Where("tick = ? ", ex.Tick0).First(card0).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	card1 := &models.Drc20Collect{}
	err = v.dbc.DB.Where("tick = ? ", ex.Tick1).First(card1).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if ex.Amt0.Int().Cmp(Number0) < 1 || ex.Amt1.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	return nil
}

func (v *Verifys) VerifyExchangeTrade(ex *models.ExchangeInfo) error {
	if ex.Amt1.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	exc := &models.ExchangeCollect{}
	err := v.dbc.DB.Where("ex_id = ? ", ex.ExId).First(exc).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if exc.HolderAddress == ex.HolderAddress {
		return fmt.Errorf("the same address cannot be traded")
	}

	return nil
}

func (v *Verifys) VerifyExchangeCancel(ex *models.ExchangeInfo) error {
	if ex.Amt0.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	exc := &models.ExchangeCollect{}
	err := v.dbc.DB.Where("ex_id = ? ", ex.ExId).First(exc).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	return nil
}

func (v *Verifys) VerifyFileExchange(ex *models.FileExchangeInfo) error {

	switch ex.Op {
	case "create":
		return v.VerifyFileExchangeCreate(ex)
	case "trade":
		return v.VerifyFileExchangeTrade(ex)
	case "cancel":
		return v.VerifyFileExchangeCancel(ex)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) VerifyFileExchangeCreate(ex *models.FileExchangeInfo) error {

	card := &models.Drc20Collect{}
	err := v.dbc.DB.Where("tick = ? ", ex.Tick).First(card).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if ex.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	file := &models.FileCollectAddress{}
	err = v.dbc.DB.Where("file_id = ? and holder_address = ? ", ex.FileId, ex.HolderAddress).First(file).Error
	if err != nil {
		//if errors.Is(err, gorm.ErrRecordNotFound) {
		//	nft := &models.NftCollectAddress{}
		//	err = v.dbc.DB.Where("deploy_hash = ?", ex.FileId).First(nft).Error
		//	if err != nil {
		//		return fmt.Errorf("the contract does not exist err %s", err.Error())
		//	}
		//} else {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
		//}
	}

	return nil
}

func (v *Verifys) VerifyFileExchangeTrade(ex *models.FileExchangeInfo) error {
	if ex.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	exc := &models.FileExchangeCollect{}
	err := v.dbc.DB.Where("ex_id = ? ", ex.ExId).First(exc).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if ex.Amt.Cmp(exc.Amt) != 0 {
		return fmt.Errorf("the amount of tokens is not equal")
	}

	if ex.Tick != exc.Tick {
		return fmt.Errorf("the token symbol must be different")
	}

	if exc.HolderAddress == ex.HolderAddress {
		return fmt.Errorf("the same address cannot be traded")
	}

	return nil
}

func (v *Verifys) VerifyFileExchangeCancel(ex *models.FileExchangeInfo) error {

	exc := &models.FileExchangeCollect{}
	err := v.dbc.DB.Where("ex_id = ? ", ex.ExId).First(exc).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if exc.HolderAddress != ex.HolderAddress {
		return fmt.Errorf("the same address cannot be traded")
	}

	return nil
}

func (v *Verifys) VerifyBox(box *models.BoxInfo) error {

	switch box.Op {
	case "deploy":
		return v.VerifyBoxDeploy(box)
	case "mint":
		return v.VerifyBoxMint(box)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) VerifyBoxDeploy(box *models.BoxInfo) error {

	if len(box.Tick0) < 2 || len(box.Tick0) > 8 {
		return fmt.Errorf("the token symbol must be 2 or 8 letters")
	}

	card := &models.Drc20Collect{}
	err := v.dbc.DB.Where("tick = ? ", box.Tick1).First(card).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if box.Max.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if box.LiqAmt.Int().Cmp(Number0) < 1 && box.LiqBlock < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if box.LiqAmt.Int().Cmp(Number0) > 0 && box.LiqBlock > 0 {
		return fmt.Errorf("two cannot exist at the same time")
	}

	return nil
}

func (v *Verifys) VerifyBoxMint(box *models.BoxInfo) error {

	if box.Amt1.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	boxc := &models.BoxCollect{}
	err := v.dbc.DB.Where("tick0 = ? ", box.Tick0).First(boxc).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	liqa := big.NewInt(0).Add(boxc.LiqAmtFinish.Int(), box.Amt1.Int())
	if boxc.LiqAmt.Int().Cmp(Number0) > 0 && liqa.Cmp(boxc.LiqAmt.Int()) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the maximum")
	}

	return nil
}

func (v *Verifys) VerifyCross(cross *models.CrossInfo) error {
	switch cross.Op {
	case "deploy":
		return v.VerifyCrossDeploy(cross)
	case "mint":
		return v.VerifyCrossMint(cross)
	case "burn":
		return v.VerifyCrossBurn(cross)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) VerifyCrossDeploy(cross *models.CrossInfo) error {

	if len(cross.Tick) < 2 || len(cross.Tick) > 8 {
		return fmt.Errorf("the token symbol must be 2 or 8 letters")
	}

	card := &models.Drc20Collect{}
	tick := "W" + cross.Tick + "(WRAPPED-" + cross.Tick + ")"
	err := v.dbc.DB.Where("tick = ? ", tick).First(card).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
}

func (v *Verifys) VerifyCrossMint(cross *models.CrossInfo) error {

	if cross.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	card := &models.Drc20Collect{}
	err := v.dbc.DB.Where("tick = ? ", cross.Tick).First(card).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	cc := &models.CrossCollect{}
	err = v.dbc.DB.Where("tick = ? ", cross.Tick).First(cc).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if cross.HolderAddress != cc.AdminAddress {
		return fmt.Errorf("the administrator is not the same")
	}

	return nil
}

func (v *Verifys) VerifyCrossBurn(cross *models.CrossInfo) error {

	if cross.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	cardA0 := &models.Drc20CollectAddress{}
	err := v.dbc.DB.Where("tick = ? and holder_address = ?", cross.Tick, cross.HolderAddress).First(cardA0).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if cross.Amt.Cmp(cardA0.AmtSum) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) VerifyMeme20(meme20 *models.Meme20Info) error {
	switch meme20.Op {
	case "deploy":
		return v.verifyMeme20Deploy(meme20)
	case "transfer":
		return v.verifyMeme20Transfer(meme20)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifyMeme20Deploy(meme20 *models.Meme20Info) error {

	if len(meme20.Tick) == 0 || len(meme20.Name) == 0 {
		return fmt.Errorf("the token symbol and name must be 1 letters")
	}

	if meme20.Max.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if meme20.Max.Int().Cmp(maxAllowedValue) > 0 {
		return fmt.Errorf("the maximum value cannot be greater 0xffffffffffffffffffffffffffffffffffffffff")
	}

	return nil
}

func (v *Verifys) verifyMeme20Transfer(meme20 *models.Meme20Info) error {

	if meme20.Amt.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	meme1 := &models.Meme20CollectAddress{}
	err := v.dbc.DB.Where("tick_id = ? and holder_address = ?", meme20.TickId, meme20.HolderAddress).First(meme1).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist")
	}

	infosp := make([]*models.PumpInfo, 0)
	err = v.dbc.DB.Where("block_number = ? and holder_address = ? and op='trade'  and order_status = 0", meme20.BlockNumber, meme20.HolderAddress).Find(&infosp).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist")
	}

	if len(infosp) != 0 {
		return fmt.Errorf("the contract has been deployed")
	}

	infoss := make([]*models.SwapV2Info, 0)
	err = v.dbc.DB.Where("block_number = ? and holder_address = ? and op = 'swap' and order_status = 0 ", meme20.BlockNumber, meme20.HolderAddress).Find(&infoss).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist")
	}

	if len(infoss) != 0 {
		return fmt.Errorf("the contract has been deployed")
	}

	if meme20.Amt.Cmp(meme1.Amt) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}
	return nil
}

func (v *Verifys) VerifySwapV2(tx *gorm.DB, swap *models.SwapV2Info) error {
	switch swap.Op {
	case "create":
		return v.verifySwapV2Create(tx, swap)
	case "add":
		return v.verifySwapV2Add(tx, swap)
	case "remove":
		return v.verifySwapV2Remove(tx, swap)
	case "swap":
		return v.verifySwapV2Exec(tx, swap)
	}
	return nil
}

func (v *Verifys) verifySwapV2Create(tx *gorm.DB, swap *models.SwapV2Info) error {

	if swap.Amt0.Int().Cmp(Number0) < 1 || swap.Amt1.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	if swap.Tick0Id == swap.Tick1Id {
		return fmt.Errorf("the token symbol must be different")
	}

	if len(swap.Tick0Id) < MEMETICKID_LENGTH {
		cardA0 := &models.Drc20CollectAddress{}
		err := tx.Where("tick = ? and holder_address = ?", swap.Tick0Id, swap.HolderAddress).First(cardA0).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		if swap.Amt0.Cmp(cardA0.AmtSum) > 0 {
			return fmt.Errorf("the amount of tokens exceeds the balance")
		}
	} else {
		cardA0 := &models.Meme20CollectAddress{}
		err := tx.Where("tick_id = ? and holder_address = ?", swap.Tick0Id, swap.HolderAddress).First(cardA0).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		if swap.Amt0.Cmp(cardA0.Amt) > 0 {
			return fmt.Errorf("the amount of tokens exceeds the balance")
		}
	}

	if len(swap.Tick1Id) < MEMETICKID_LENGTH {
		cardA1 := &models.Drc20CollectAddress{}
		err := tx.Where("tick = ? and holder_address = ?", swap.Tick1Id, swap.HolderAddress).First(cardA1).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		if swap.Amt1.Cmp(cardA1.AmtSum) > 0 {
			return fmt.Errorf("the amount of tokens exceeds the balance")
		}
	} else {
		cardA1 := &models.Meme20CollectAddress{}
		err := tx.Where("tick_id = ? and holder_address = ?", swap.Tick1Id, swap.HolderAddress).First(cardA1).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		if swap.Amt1.Cmp(cardA1.Amt) > 0 {
			return fmt.Errorf("the amount of tokens exceeds the balance")
		}
	}

	return nil
}

func (v *Verifys) verifySwapV2Add(tx *gorm.DB, swap *models.SwapV2Info) error {

	if swap.Amt0.Int().Cmp(Number0) < 1 || swap.Amt1.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	swapLiquidity := &models.SwapV2Liquidity{}
	err := tx.Where("pair_id = ?", swap.PairId).First(swapLiquidity).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	sum0 := models.NewNumber(0)
	sum1 := models.NewNumber(0)

	if len(swapLiquidity.Tick0Id) < MEMETICKID_LENGTH {
		cardA0 := &models.Drc20CollectAddress{}
		err = tx.Where("tick = ? and holder_address = ?", swapLiquidity.Tick0Id, swap.HolderAddress).First(cardA0).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		sum0 = cardA0.AmtSum
	} else {
		cardA0 := &models.Meme20CollectAddress{}
		err = tx.Where("tick_id = ? and holder_address = ?", swapLiquidity.Tick0Id, swap.HolderAddress).First(cardA0).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		sum0 = cardA0.Amt
	}

	if len(swapLiquidity.Tick1Id) < MEMETICKID_LENGTH {
		cardA1 := &models.Drc20CollectAddress{}
		err = tx.Where("tick = ? and holder_address = ?", swapLiquidity.Tick1Id, swap.HolderAddress).First(cardA1).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		sum1 = cardA1.AmtSum
	} else {
		cardA1 := &models.Meme20CollectAddress{}
		err = tx.Where("tick_id = ? and holder_address = ?", swapLiquidity.Tick1Id, swap.HolderAddress).First(cardA1).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}

		sum1 = cardA1.Amt
	}

	amountBOptimal := big.NewInt(0).Mul(swap.Amt0.Int(), swapLiquidity.Amt1.Int())

	if amountBOptimal.Cmp(big.NewInt(0)) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	amountBOptimal = big.NewInt(0).Div(amountBOptimal, swapLiquidity.Amt0.Int())

	if amountBOptimal.Cmp(swap.Amt1Min.Int()) < 0 {
		amountAOptimal := big.NewInt(0).Mul(swap.Amt1.Int(), swapLiquidity.Amt0.Int())
		if amountAOptimal.Cmp(big.NewInt(0)) < 1 {
			return fmt.Errorf("the amount of tokens exceeds the 0")
		}
		amountAOptimal = big.NewInt(0).Div(amountAOptimal, swapLiquidity.Amt1.Int())

		if amountAOptimal.Cmp(swap.Amt0Min.Int()) < 0 {
			return fmt.Errorf("the amount of tokens exceeds the min")
		} else {
			if amountAOptimal.Cmp(sum0.Int()) > 0 {
				return fmt.Errorf("the amount of tokens exceeds the balance")
			}

			if swap.Amt1.Cmp(sum1) > 0 {
				return fmt.Errorf("the amount of tokens exceeds the balance")
			}
		}
	} else {
		if swap.Amt0.Cmp(sum0) > 0 {
			return fmt.Errorf("the amount of tokens exceeds the balance")
		}

		if amountBOptimal.Cmp(sum1.Int()) > 0 {
			return fmt.Errorf("the amount of tokens exceeds the max")
		}
	}

	return nil
}

func (v *Verifys) verifySwapV2Remove(tx *gorm.DB, swap *models.SwapV2Info) error {

	if swap.Liquidity.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	swapLiquidity := &models.SwapV2Liquidity{}
	err := tx.Where("pair_id = ?", swap.PairId).First(swapLiquidity).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if swap.Liquidity.Int().Cmp(swapLiquidity.LiquidityTotal.Int()) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	cardA0 := &models.Meme20CollectAddress{}
	err = tx.Where("tick_id = ? and holder_address = ?", swapLiquidity.PairId, swap.HolderAddress).First(cardA0).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	sum0 := cardA0.Amt

	if swap.Liquidity.Cmp(sum0) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance")
	}

	return nil
}

func (v *Verifys) verifySwapV2Exec(tx *gorm.DB, swap *models.SwapV2Info) error {

	if swap.Amt0.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	swapLiquidity := &models.SwapV2Liquidity{}
	err := tx.Where("pair_id = ?", swap.PairId).First(swapLiquidity).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	infos := make([]*models.SwapV2Info, 0)
	err = tx.Where("block_number = ? and holder_address = ? and op = 'swap' and order_status = 0", swap.BlockNumber, swap.HolderAddress).Find(&infos).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist")
	}

	if len(infos) != 0 {
		return fmt.Errorf("the contract has been deployed")
	}

	if swap.Tick0Id == swapLiquidity.Tick0Id {
		swap.Tick0 = swapLiquidity.Tick0
		swap.Tick1 = swapLiquidity.Tick1
		swap.Tick1Id = swapLiquidity.Tick1Id
	} else if swap.Tick0Id == swapLiquidity.Tick1Id {
		swap.Tick1Id = swapLiquidity.Tick0Id
		swap.Tick0 = swapLiquidity.Tick1
		swap.Tick1 = swapLiquidity.Tick0
	} else {
		return fmt.Errorf("the contract does not exist err")
	}

	amtMap := make(map[string]*big.Int)
	amtMap[swapLiquidity.Tick0Id] = swapLiquidity.Amt0.Int()
	amtMap[swapLiquidity.Tick1Id] = swapLiquidity.Amt1.Int()

	amtfee0 := new(big.Int).Div(swap.Amt0.Int(), big.NewInt(1000))
	amtin := new(big.Int).Mul(amtfee0, big.NewInt(3))
	amtin = new(big.Int).Sub(swap.Amt0.Int(), amtin)

	amtout := new(big.Int).Mul(amtin, amtMap[swap.Tick1Id])
	amtout = new(big.Int).Div(amtout, new(big.Int).Add(amtMap[swap.Tick0Id], amtin))

	if amtout.Cmp(swap.Amt1Min.Int()) < 0 {
		return fmt.Errorf("the minimum output less than the limit output amount %s", amtout.String())
	}

	sum0 := models.NewNumber(0)

	if len(swap.Tick0Id) < MEMETICKID_LENGTH {
		cardA0 := &models.Drc20CollectAddress{}
		err = tx.Where("tick = ? and holder_address = ?", swap.Tick0Id, swap.HolderAddress).First(cardA0).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}
		sum0 = cardA0.AmtSum
	} else {
		cardA0 := &models.Meme20CollectAddress{}
		err = tx.Where("tick_id = ? and holder_address = ?", swap.Tick0Id, swap.HolderAddress).First(cardA0).Error
		if err != nil {
			return fmt.Errorf("the contract does not exist err %s", err.Error())
		}
		sum0 = cardA0.Amt
	}

	if swap.Amt0.Cmp(sum0) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the balance of the input token")
	}

	return nil
}

func (v *Verifys) VerifyPump(tx *gorm.DB, pump *models.PumpInfo) error {
	switch pump.Op {
	case "deploy":
		return v.verifyPumpDeploy(tx, pump)
	case "trade":
		return v.verifyPumpTrade(tx, pump)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifyPumpDeploy(tx *gorm.DB, pump *models.PumpInfo) error {

	// pump.Reserve 0-99
	if pump.Reserve < 0 || pump.Reserve > 99 {
		return fmt.Errorf("the reserve must be between 0 and 99")
	}

	if pump.Reserve > 0 && pump.Amt1.Int().Cmp(Number0) > 0 {
		return fmt.Errorf("the reserve and the amount cannot exist at the same time")
	}

	if pump.Tick1Id != "WDOGE(WRAPPED-DOGE)" {
		return fmt.Errorf("the token symbol must be WDOGE(WRAPPED-DOGE)")
	}

	pumpl := &models.PumpLiquidity{}
	err := tx.Where("tick0_id = ? ", pump.Tick0Id).First(pumpl).Error
	if err == nil {
		return fmt.Errorf("the contract has been deployed")
	}

	if pump.Amt1.Int().Cmp(big.NewInt(500000000000)) > 0 {
		return fmt.Errorf("the amount of tokens exceeds the 5000 doge")
	}

	return nil
}

func (v *Verifys) verifyPumpTrade(tx *gorm.DB, pump *models.PumpInfo) error {

	if pump.Amt0.Int().Cmp(Number0) < 1 {
		return fmt.Errorf("the amount of tokens exceeds the 0")
	}

	pumpl := &models.PumpLiquidity{}
	err := tx.Where("tick0_id = ? ", pump.PairId).First(pumpl).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist err %s", err.Error())
	}

	if pumpl.Amt1.Int().Cmp(DogeMax) >= 0 {
		return fmt.Errorf("pump inner plate has been completed")
	}

	infos := make([]*models.PumpInfo, 0)
	err = tx.Where("block_number = ? and holder_address = ? and op='trade' and order_status = 0", pump.BlockNumber, pump.HolderAddress).Find(&infos).Error
	if err != nil {
		return fmt.Errorf("the contract does not exist")
	}

	if len(infos) != 0 {
		return fmt.Errorf("the contract has been deployed")
	}

	if pump.Tick0Id == pumpl.Tick0Id {
		pump.Tick1Id = pumpl.Tick1Id
	} else if pump.Tick0Id == pumpl.Tick1Id {
		pump.Tick1Id = pumpl.Tick0Id
	} else {
		return fmt.Errorf("the contract does not exist err")
	}

	return nil
}

func (v *Verifys) VerifyInvite(invite *models.InviteInfo) error {
	switch invite.Op {
	case "deploy":
		return v.verifyInviteDeploy(invite)
	default:
		return fmt.Errorf("do not support the type of tokens")
	}
}

func (v *Verifys) verifyInviteDeploy(invite *models.InviteInfo) error {

	if invite.InviteAddress == invite.HolderAddress {
		return fmt.Errorf("the same address cannot be invited")
	}

	invitec := &models.InviteCollect{}
	err := v.dbc.DB.Where("holder_address = ? ", invite.HolderAddress).First(invitec).Error
	if err == nil {
		return fmt.Errorf("already invited")
	}

	return nil
}
