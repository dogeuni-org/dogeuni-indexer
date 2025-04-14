# dogeuni-indexer




## start up

### 1. Install and run dogecoin
Please check out [dogecoin](docs/dogecoin.md)

### 2. Compile golang program
```go
go build.
```

### 3. Download data

https://github.com/dogeuni-org/dogeuni-indexer/releases

Download the latest db data of releases and put it in the data directory

```shell
cat dogeuni.zip.* > dogeuni.zip
unzip dogeuni.zip
```

### 4. Config.json
```json
{
  "http_server": {
    "switch": false,
    "server": ":8089"
  },
  "leveldb": {
    "path": "data/leveldb"
  },
  "sqlite": {
    "switch": true,
    "database": "data/dogeuni.db"
  },
  "chain": {
    "chain_name": "dogecoin",
    "rpc": "127.0.0.1:22555",
    "user_name": "admin",
    "pass_word": "admin"
  },
  "explorer": {
    "switch": true,
    "from_block": 0
  },
  "ipfs": "",
  "debug_level": 3
}
```


### 5. Run
```go
./dogeuni-indexer
```



### Router Document
Please check out [router](https://documenter.getpostman.com/view/8337528/2s9YeN18PF)
