package models

// CardityContract represents a contract deployed via Cardity language
// This table stores the basic metadata and ABI reference so indexers and
// applications can discover and introspect contracts generically.
type CardityContract struct {
	Id           int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	ContractId   string `json:"contract_id" gorm:"uniqueIndex"`
	Protocol     string `json:"protocol" gorm:"index"`
	Version      string `json:"version"`
	AbiJSON      string `json:"abi_json" gorm:"type:text"`
	AbiCID       string `json:"abi_cid"`
	CarcHash     string `json:"carc_hash"`
	CarcSHA256   string `json:"carc_sha256" gorm:"index"`
	Size         int64  `json:"size"`
	PackageId    string `json:"package_id" gorm:"index"`
	ModuleName   string `json:"module_name" gorm:"index"`
	DeployTxHash string `json:"deploy_tx_hash" gorm:"index"`
	CarcIpfsPath string `json:"carc_ipfs_path" gorm:"type:varchar(512)"`
	Creator      string `json:"creator" gorm:"index"`
	BlockHash    string `json:"block_hash"`
	BlockNumber  int64  `json:"block_number" gorm:"index"`
	CreateDate   int64  `json:"create_date" gorm:"index"`
}

// CardityInvocationLog stores method invocations on a Cardity contract
type CardityInvocationLog struct {
	Id          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	ContractId  string `json:"contract_id" gorm:"index,uniqueIndex:uniq_cardi_inv"`
	Method      string `json:"method" gorm:"index,uniqueIndex:uniq_cardi_inv"`
	MethodFQN   string `json:"method_fqn" gorm:"index"`
	ArgsJSON    string `json:"args_json" gorm:"type:text"`
	ArgsText    string `json:"args_text" gorm:"type:varchar(256);index"`
	TxHash      string `json:"tx_hash" gorm:"index,uniqueIndex:uniq_cardi_inv"`
	BlockHash   string `json:"block_hash"`
	BlockNumber int64  `json:"block_number" gorm:"index"`
	CreateDate  int64  `json:"create_date" gorm:"index"`
}

// CardityEventLog stores events emitted by a Cardity contract
type CardityEventLog struct {
	Id          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	ContractId  string `json:"contract_id" gorm:"index,uniqueIndex:uniq_cardi_evt"`
	EventName   string `json:"event_name" gorm:"index,uniqueIndex:uniq_cardi_evt"`
	ParamsJSON  string `json:"params_json" gorm:"type:text"`
	TxHash      string `json:"tx_hash" gorm:"index,uniqueIndex:uniq_cardi_evt"`
	BlockHash   string `json:"block_hash"`
	BlockNumber int64  `json:"block_number" gorm:"index"`
	CreateDate  int64  `json:"create_date" gorm:"index"`
}

// CardityBackfillState stores progress checkpoints for resumable backfill
type CardityBackfillState struct {
	Id    int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name  string `json:"name" gorm:"uniqueIndex"`
	Value string `json:"value" gorm:"type:text"`
}

// CardityPackage represents a multi-module package deployment
type CardityPackage struct {
	Id           int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	PackageId    string `json:"package_id" gorm:"uniqueIndex"`
	Version      string `json:"version"`
	PackageABI   string `json:"package_abi" gorm:"type:text"`
	ModulesJSON  string `json:"modules_json" gorm:"type:text"`
	DeployTxHash string `json:"deploy_tx_hash" gorm:"index"`
	CarcIpfsPath string `json:"carc_ipfs_path" gorm:"type:varchar(512)"`
	BlockHash    string `json:"block_hash"`
	BlockNumber  int64  `json:"block_number" gorm:"index"`
	CreateDate   int64  `json:"create_date" gorm:"index"`
}

// CardityCollectAddress tracks ownership similar to file_collect_address
type CardityCollectAddress struct {
	Id            int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	ContractId    string `json:"contract_id" gorm:"index"`
	PackageId     string `json:"package_id" gorm:"index"`
	ModuleName    string `json:"module_name" gorm:"index"`
	HolderAddress string `json:"holder_address" gorm:"index"`
	CarcIpfsPath  string `json:"carc_ipfs_path" gorm:"type:varchar(512)"`
	CreateDate    int64  `json:"create_date" gorm:"index"`
}

// CardityRevert for potential future ownership reverts
type CardityRevert struct {
	Id          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	ContractId  string `json:"contract_id"`
	PackageId   string `json:"package_id"`
	ModuleName  string `json:"module_name"`
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	TxHash      string `json:"tx_hash"`
	BlockNumber int64  `json:"block_number"`
	CreateDate  int64  `json:"create_date"`
}

// CardityModule represents a single module within a package
type CardityModule struct {
	Id           int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	PackageId    string `json:"package_id" gorm:"index"`
	Name         string `json:"name" gorm:"index"`
	AbiJSON      string `json:"abi_json" gorm:"type:text"`
	CarcB64      string `json:"carc_b64" gorm:"type:text"`
	CarcSHA256   string `json:"carc_sha256" gorm:"index"`
	Size         int64  `json:"size"`
	DeployTxHash string `json:"deploy_tx_hash" gorm:"index"`
	BlockHash    string `json:"block_hash"`
	BlockNumber  int64  `json:"block_number" gorm:"index"`
	CreateDate   int64  `json:"create_date" gorm:"index"`
}

// CardityBundlePart stores sharded package deployment parts for reassembly
type CardityBundlePart struct {
	Id         int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	BundleId   string `json:"bundle_id" gorm:"index,uniqueIndex:uniq_bundle_idx"`
	Idx        int    `json:"idx" gorm:"uniqueIndex:uniq_bundle_idx"`
	Total      int    `json:"total"`
	PackageId  string `json:"package_id"`
	Version    string `json:"version"`
	ModuleName string `json:"module_name"`
	AbiJSON    string `json:"abi_json" gorm:"type:text"`
	CarcB64    string `json:"carc_b64" gorm:"type:text"`
	TxHash     string `json:"tx_hash" gorm:"index"`
	CreateDate int64  `json:"create_date" gorm:"index"`
}
