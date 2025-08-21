## dogeuni-indexer

High-performance Dogecoin on-chain indexer and HTTP API for multiple protocols (DRC-20, Meme20, Swap, Exchange, NFT, WDOGE, File, Cross, Box, Stake, Pump, Invite) plus a generic Cardity smart-contract indexing layer.

### Features
- DRC-20, Meme20 and more protocols indexing and query APIs
- Cardity generic indexing (deploy/invoke logs and events) with auto-migrated tables
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

### Cardity generic indexing (MVP)
- Supports inscriptions with envelope `p=cardity` and `op=deploy|invoke`
- Stores:
  - CardityContract (deploy metadata & ABI reference)
  - CardityInvocationLog (method calls)
  - CardityEventLog (events; runtime integration planned)
- Tables are auto-migrated on startup

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

