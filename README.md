## dogeuni-indexer

High-performance Dogecoin on-chain indexer and HTTP API for multiple protocols (DRC-20, Meme20, Swap, Exchange, NFT, WDOGE, File, Cross, Box, Stake, Pump, Invite) plus a generic Cardity smart-contract indexing layer.

### Features
- DRC-20, Meme20 and more protocols indexing and query APIs
- Cardity generic indexing:
  - Supports deploy, deploy_package, deploy_part, invoke
  - Package/Module indexing with shard reassembly (bundle_id + idx/total)
  - Records CARC SHA256/hash and size; optional upload path (future IPFS/S3)
  - Resumable backfill tool and daemon script
- Dual storage: SQLite for snapshot/query, MySQL supported for writes; LevelDB for caches
- HTTP APIs via Gin; Postman doc available

### Quick start (query only)
Suitable if you just want to run read APIs without syncing a Dogecoin node.

1) Download the prebuilt SQLite snapshot
- Releases: `https://github.com/dogeuni-org/dogeuni-indexer/releases`
- Place the merged DB file at `data/dogeuni.db` (or project root)
```bash
cat dogeuni.zip.* > dogeuni.zip
unzip dogeuni.zip    # produces data/dogeuni.db
```

2) Create `config.json` in project root
```json
{
  "http_server": { "switch": true, "server": ":8089" },
  "leveldb": { "path": "data/leveldb" },
  "sqlite": { "switch": true, "database": "data/dogeuni.db" },
  "mysql": { "switch": false, "server": "127.0.0.1", "port": 3306, "user_name": "root", "pass_word": "root", "database": "dogeuni" },
  "chain": { "chain_name": "dogecoin", "rpc": "127.0.0.1:22555", "user_name": "admin", "pass_word": "admin" },
  "explorer": { "switch": false, "from_block": 0 },
  "cardity": { "enable": true, "runtime_path": "", "abi_registry": "" },
  "ipfs": "",  
  "debug_level": 3
}
```

3) Build and run
```bash
go build -o dogeuni-indexer .
./dogeuni-indexer
```

4) Health check
```bash
curl -s -X POST http://127.0.0.1:8089/v3/info/lastnumber
curl -s -X POST http://127.0.0.1:8089/v4/info/lastnumber
```

### MySQL mode and full indexing (optional)
Use MySQL if you want to run the explorer (write path) and persist live chain updates. Note: v3 market/summary endpoints still read from SQLite snapshot.

1) Prepare MySQL database
```bash
mysql -uroot -proot -e "CREATE DATABASE IF NOT EXISTS dogeuni DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;"
```

2) Sample `config.json` (MySQL on, explorer off)
```json
{
  "http_server": { "switch": true, "server": ":8089" },
  "leveldb": { "path": "data/leveldb" },
  "sqlite": { "switch": false, "database": "data/dogeuni.db" },
  "mysql": { "switch": true, "server": "127.0.0.1", "port": 3306, "user_name": "root", "pass_word": "root", "database": "dogeuni" },
  "chain": { "chain_name": "dogecoin", "rpc": "127.0.0.1:22555", "user_name": "admin", "pass_word": "admin" },
  "explorer": { "switch": false, "from_block": 0 },
  "cardity": { "enable": true, "runtime_path": "", "abi_registry": "" },
  "ipfs": "",
  "debug_level": 3
}
```

3) Enable explorer (requires Dogecoin full node RPC)
- Set `"explorer.switch": true`
- Ensure `dogecoin.conf` RPC matches `chain.*` in config
- The indexer auto-migrates Cardity tables; domain tables (DRC-20, etc.) must already exist (or use SQLite snapshot for v3 queries)

### APIs
- Postman documentation: `https://documenter.getpostman.com/view/8337528/2s9YeN18PF`
- Notable endpoints
  - v3 info: `POST /v3/info/lastnumber`
  - v4 info: `POST /v4/info/lastnumber`, `POST /v4/info/blocknumber`
  - v4 drc20: `POST /v4/drc20/order`, `/collect`, `/collect-address`, `/history`
  - v4 meme20: `POST /v4/meme20/order`, `/collect`, `/collect-address`, `/history`
  - v4 cardity (generic):
    - `POST /v4/cardity/contracts`
    - `POST /v4/cardity/invocations`
    - `POST /v4/cardity/events`
    - `POST /v4/cardity/packages`
    - `POST /v4/cardity/modules`

### Cardity generic indexing
- Accepted payloads
  - JSON envelope: `p=cardity`, `op in {deploy, deploy_package, deploy_part, invoke}`; `abi` can be JSON or string; CARC from `carc_b64` or `file_hex/file_b64`.
  - Shards: `bundle_id`, `idx`, `total` for `deploy_part`; reassembles into modules when all parts arrive.
  - Method FQN: if `module` provided and `method` has no dot, stored as `module.method`.
- Stored entities
  - CardityContract: `contract_id, protocol, version, abi_json, carc_sha256, size, package_id, module_name, deploy_tx_hash`
  - CardityPackage: `package_id, version, package_abi, modules_json, deploy_tx_hash`
  - CardityModule: `package_id, name, abi_json, carc_b64, carc_sha256, size, deploy_tx_hash`
  - CardityInvocationLog: `contract_id, method, method_fqn, args_json, args_text, tx_hash, block_number`
  - CardityEventLog: reserved for runtime integration
- Tables are auto-migrated on startup

### Backfill (resumable)
- One-off CLI
```bash
go build -o backfill ./cmd/backfill
./backfill -target all -batch 2000 -config config.json
```
- Daemon (resumable with checkpoints)
```bash
./scripts/backfill_daemon.sh all 8000 config.json
# tail logs
tail -f logs/backfill_*.log
# stop
kill $(cat logs/backfill.pid)
```
- Checkpoints are saved in `CardityBackfillState`; safe to restart and resume.

### Troubleshooting
- Error 1146: table 'dogeuni.block' doesn't exist
  - Cause: Using MySQL without domain tables
  - Fix: Set `sqlite.switch=true` and provide `data/dogeuni.db` for v3 queries; or import/create MySQL schema for domain tables
- v3 endpoints return 500
  - Ensure `sqlite.database` points to a valid snapshot file
- Change HTTP port
  - Edit `http_server.server` (e.g., `":8090"`)

### Build from source
```bash
go build -o dogeuni-indexer .
```

### Docs
- Dogecoin setup: `docs/dogecoin.md`

