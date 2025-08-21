package storage

import "dogeuni-indexer/models"

// AutoMigrateCardity ensures Cardity tables exist
func (db *DBClient) AutoMigrateCardity() error {
	return db.DB.AutoMigrate(
		&models.CardityContract{},
		&models.CardityInvocationLog{},
		&models.CardityEventLog{},
		&models.CardityBackfillState{},
		&models.CardityPackage{},
		&models.CardityModule{},
		&models.CardityBundlePart{},
		&models.CardityCollectAddress{},
		&models.CardityRevert{},
	)
}
