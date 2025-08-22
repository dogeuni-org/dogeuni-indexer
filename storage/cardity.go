package storage

import (
	"dogeuni-indexer/models"

	"gorm.io/gorm/clause"
)

// SaveCardityContract persists a CardityContract. If the contract_id already exists,
// it will be updated with the latest metadata.
func (db *DBClient) SaveCardityContract(contract *models.CardityContract) error {
	return db.DB.Save(contract).Error
}

// SaveCardityInvocation stores a method invocation log
func (db *DBClient) SaveCardityInvocation(inv *models.CardityInvocationLog) error {
	// idempotent on unique tx_hash
	return db.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(inv).Error
}

// SaveCardityEvents stores multiple event logs in batch
func (db *DBClient) SaveCardityEvents(events []*models.CardityEventLog) error {
	if len(events) == 0 {
		return nil
	}
	return db.DB.Create(&events).Error
}

// Package and Module persistence
func (db *DBClient) SaveCardityPackage(pkg *models.CardityPackage) error {
	return db.DB.Save(pkg).Error
}

func (db *DBClient) SaveCardityModule(mod *models.CardityModule) error {
	return db.DB.Save(mod).Error
}

func (db *DBClient) SaveBundlePart(part *models.CardityBundlePart) error {
	// upsert per (bundle_id, idx)
	return db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "bundle_id"}, {Name: "idx"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"carc_b64": part.CarcB64, "abi_json": part.AbiJSON, "package_id": part.PackageId, "version": part.Version, "module_name": part.ModuleName, "tx_hash": part.TxHash, "block_hash": part.BlockHash, "block_number": part.BlockNumber, "create_date": part.CreateDate}),
	}).Create(part).Error
}

func (db *DBClient) FindBundleParts(bundleId string) ([]*models.CardityBundlePart, error) {
	parts := make([]*models.CardityBundlePart, 0)
	err := db.DB.Where("bundle_id = ?", bundleId).Order("idx asc").Find(&parts).Error
	return parts, err
}
