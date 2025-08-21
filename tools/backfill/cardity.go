package backfill

import (
	"dogeuni-indexer/models"
	"dogeuni-indexer/storage"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// BackfillDRC20 scans historical DRC-20 records and mirrors them into Cardity tables.
func BackfillDRC20(db *storage.DBClient, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1000
	}

	// Contracts: ensure unique ticks from deploy
	ticks := make([]string, 0)
	if err := db.DB.Model(&models.Drc20Info{}).Where("op = ?", "deploy").Distinct().Pluck("tick", &ticks).Error; err != nil {
		return err
	}
	for _, t := range ticks {
		tUpper := strings.ToUpper(t)
		_ = db.SaveCardityContract(&models.CardityContract{
			ContractId: tUpper,
			Protocol:   "DRC20",
			Version:    "1.0",
		})
	}

	// Invocations: mint / transfer (resumable by offset)
	// read offset state
	state := &models.CardityBackfillState{}
	_ = db.DB.Where("name = ?", "drc20_offset").First(state).Error
	offset := 0
	if v, err := strconv.Atoi(state.Value); err == nil {
		offset = v
	}
	for {
		list := make([]*models.Drc20Info, 0)
		if err := db.DB.Where("op in ?", []string{"mint", "transfer"}).Order("id asc").Limit(batchSize).Offset(offset).Find(&list).Error; err != nil {
			return err
		}
		if len(list) == 0 {
			break
		}

		for _, d := range list {
			contractId := strings.ToUpper(d.Tick)
			var (
				method string
				args   []interface{}
			)
			switch strings.ToLower(d.Op) {
			case "mint":
				method = "mint"
				args = []interface{}{d.HolderAddress, d.Amt.Int().String(), contractId}
			case "transfer":
				method = "transfer"
				args = []interface{}{d.HolderAddress, d.ToAddress, d.Amt.Int().String(), contractId}
			default:
				continue
			}
			ab, _ := json.Marshal(args)
			_ = db.SaveCardityInvocation(&models.CardityInvocationLog{
				ContractId:  contractId,
				Method:      method,
				ArgsJSON:    string(ab),
				TxHash:      d.TxHash,
				BlockHash:   d.BlockHash,
				BlockNumber: d.BlockNumber,
			})
		}
		offset += len(list)
		_ = db.DB.Save(&models.CardityBackfillState{Name: "drc20_offset", Value: strconv.Itoa(offset)}).Error
		if len(list) < batchSize {
			break
		}
	}
	return nil
}

// BackfillMeme20 mirrors Meme20 history into Cardity tables.
func BackfillMeme20(db *storage.DBClient, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1000
	}

	// Contracts: from deploy
	ids := make([]string, 0)
	if err := db.DB.Model(&models.Meme20Info{}).Where("op = ?", "deploy").Distinct().Pluck("tick_id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		_ = db.SaveCardityContract(&models.CardityContract{
			ContractId: id,
			Protocol:   "MEME20",
			Version:    "1.0",
		})
	}

	// Invocations: transfer (resumable by offset)
	state := &models.CardityBackfillState{}
	_ = db.DB.Where("name = ?", "meme20_offset").First(state).Error
	offset := 0
	if v, err := strconv.Atoi(state.Value); err == nil {
		offset = v
	}
	for {
		list := make([]*models.Meme20Info, 0)
		if err := db.DB.Where("op = ?", "transfer").Order("id asc").Limit(batchSize).Offset(offset).Find(&list).Error; err != nil {
			return err
		}
		if len(list) == 0 {
			break
		}
		for _, m := range list {
			args, _ := json.Marshal([]interface{}{m.HolderAddress, m.ToAddress, m.Amt.Int().String(), m.TickId})
			_ = db.SaveCardityInvocation(&models.CardityInvocationLog{
				ContractId:  m.TickId,
				Method:      "transfer",
				ArgsJSON:    string(args),
				TxHash:      m.TxHash,
				BlockHash:   m.BlockHash,
				BlockNumber: m.BlockNumber,
			})
		}
		offset += len(list)
		_ = db.DB.Save(&models.CardityBackfillState{Name: "meme20_offset", Value: strconv.Itoa(offset)}).Error
		if len(list) < batchSize {
			break
		}
	}
	return nil
}

// BackfillAll runs both DRC-20 and Meme20 backfills
func BackfillAll(db *storage.DBClient, batchSize int) error {
	if err := BackfillDRC20(db, batchSize); err != nil {
		return fmt.Errorf("drc20: %w", err)
	}
	if err := BackfillMeme20(db, batchSize); err != nil {
		return fmt.Errorf("meme20: %w", err)
	}
	return nil
}
