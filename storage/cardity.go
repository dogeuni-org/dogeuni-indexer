package storage

import (
	"dogeuni-indexer/models"
)

// SaveCardityContract persists a CardityContract. If the contract_id already exists,
// it will be updated with the latest metadata.
func (db *DBClient) SaveCardityContract(contract *models.CardityContract) error {
	return db.DB.Save(contract).Error
}

// SaveCardityInvocation stores a method invocation log
func (db *DBClient) SaveCardityInvocation(inv *models.CardityInvocationLog) error {
	return db.DB.Create(inv).Error
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
	return db.DB.Create(part).Error
}

func (db *DBClient) FindBundleParts(bundleId string) ([]*models.CardityBundlePart, error) {
	parts := make([]*models.CardityBundlePart, 0)
	err := db.DB.Where("bundle_id = ?", bundleId).Order("idx asc").Find(&parts).Error
	return parts, err
}
