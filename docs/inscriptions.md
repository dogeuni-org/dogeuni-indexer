# drc-20

## deploy
```json lines
// deploy
{ 
  "p": "drc-20",
  "op": "deploy",
  "tick": "unix",
  "max": "10", 
  "amt": "10",
  "lim": "10",
  "dec": "8",
  "burn": "",
  "func": ""
}

// mint
{ 
  "p": "drc-20",
  "op": "mint",
  "tick": "unix",
  "amt": "10"
}

// transfer
{ 
  "p": "drc-20",
  "op": "deploy",
  "tick": "unix",
  "amt": "10"
}
```

## pair-v1

```json lines
// create
{ 
  "p": "pair-v1",
  "op": "create", 
  "tick0": "CZZ",
  "tick1": "UNIX",
  "amt0": "1000",
  "amt1": "1000",
  "doge": 0
}

// add
{ 
  "p": "pair-v1",
  "op": "add",
  "tick0": "CZZ",
  "tick1": "UNIX",
  "amt0": "1000",
  "amt1": "1000",
  "amt0_min": "1000",
  "amt1_min": "1000",
  "doge": 0
}

// remove
{ 
  "p": "pair-v1",
  "op": "remove",
  "tick0": "CZZ",
  "tick1": "UNIX",
  "liquidity": "10", 
  "doge": 0
}

// swap
{ 
  "p": "pair-v1",
  "op": "swap",
  "tick0": "CZZ",
  "tick1": "UNIX",
  "amt0": "1000",
  "amt1_min": "1000",
  "doge": 0
}
```

## wdoge

```json lines lines

// deposit
{ 
	"p": "wdoge",
  "op": "deposit",
  "amt": "100",
}

// withdraw
{ 
 	"p": "wdoge",
  "op": "withdraw",
  "amt": "100",
}

```

## file

```json lines lines

// deploy
{ 
}

// transfer
{ 
  "p": "file",
  "op": "transfer", 
  "file_id": ""
}

```

## stake-v1

```json lines

// stake
{ 
  "p": "stake-v1",
  "op": "stake",
  "tick": "unix", 
  "amt": "10", 
}

{ 
  "p": "stake-v1",
  "op": "unstake", 
  "tick": "unix-doge", 
  "amt": "10", 
}

{ 
  "p": "stake-v1",
  "op": "getallreward",
  "tick": "unix-doge"
}

```

## order-v1

```json lines lines

// create
{ 
  "p": "order-v1",
  "op": "create", 
  "tick0": "unix",
  "tick1": "wdoge",
  "amt0":"10000000",
  "amt1":"10000000"
}

// trade
{ 
  "p": "order-v1",
  "op": "trade", 
  "exid": "asdasdasdasdas",
  "amt1":"10000000"
}

// cancel
{ 
  "p": "order-v2",
  "op": "cancel", 
  "exid": "unix",
  "amt0":"10000000"
}
```

## order-v2

```json lines lines

// create
{ 
  "p": "order-v2",
  "op": "create", 
  "file_id": "unix",
  "tick": "wdoge",
  "amt":"10000000"
}

// trade
{ 
  "p": "order-v2",
  "op": "trade", 
  "ex_id": "unix",
  "tick": "wdoge",
  "amt":"10000000"
}

// cancel
{ 
  "p": "order-v2",
  "op": "cancel", 
  "ex_id": "unix"
}
```

## cross-v1

```json lines lines

// deploy
{ 
  "p": "cross",
  "op": "deploy", 
  "chain":"CZZ",
  "tick": "UNIX",
  "admin_address":"DTZSTXecLmSXpRGSfht4tAMyqra1wsL7xb"  
}

// mint
{ 
  "p": "cross",
  "op": "mint",
  "chain":"CZZ",
  "tick": "UNIX",
  "amt": "10", 
  "to_address":""
}

// burn
{ 
  "p": "cross",
  "op": "burn",
  "chain":"CZZ",
  "tick": "UNIX",
  "amt": "10", 
}

```

## box-v1

```json lines lines

// deploy
{ 
  "p": "box-v1",
  "op": "transfer", 
  "tick0": "",
  "tick1": "",
  "max": "",
  "amt0": "",
  "liqamt": "",
  "liqblock": "",
  "amt1": ""
}

// mint
{ 
  "p": "box-v1",
  "op": "mint", 
  "tick0": "",
  "amt1":"123"
}

```

## stake-v2

```json lines lines

// create
{ 
  "p": "stake-v2",
  "op": "create", 
  "tick0": "unix-swap-wdoge",
  "tick1": "wdoge",
  "reward":"10000000", 
  "each_reward":"1000",   
  "lock_block": 1000    
}

// stake
{ 
  "p": "stake-v2",
  "op": "stake",
  "stake_id": "asdasdasd",
  "amt": "10", 
}

// unstake
{ 
  "p": "stake-v2",
  "op": "unstake",
  "stake_id": "asdasdasd",
  "amt": "10", 
}

// getreward
{ 
  "p": "stake-v2",
  "op": "getreward",
  "stake_id": "asdasdasd"
}

```

# pump
```json lines lines

// deploy
{ 
  "p": "pump",
  "op": "deploy",
  "tick": "CARDI",
  "amt": "10",
  "symbol": "MEME",
  "name": "meme name",
  "logo": "path",
  "reserve":10,  
  "doge": 1
}

// trade
{ 
  "p": "pump",
  "op": "trade",
  "pair_id": "xxxxxxx",
  "tick0_id": "CZZ",
  "amt0": "1000",
  "amt1_min": "1000",
  "doge": 1
}
```

## meme-20

```json lines lines

// deploy
{ 
  "p": "meme-20",
  "op": "deploy",
  "tick": "unix",
  "name": "unix",
  "max": "10"
}

// transfer
{ 
  "p": "meme-20",
  "op": "transfer",
  "tick_id": "unix",
  "amt": "10"
}

```

## pair-v2

```json lines
// create
{ 
  "p": "pair-v2",
  "op": "create", 
  "tick0_id": "CZZ",
  "tick1_id": "UNIX",
  "amt0": "1000",
  "amt1": "1000",
  "doge": 0
}

// add
{ 
  "p": "pair-v2",
  "op": "add",
  "pair_id":"",
  "tick0_id": "CZZ",
  "tick1_id": "UNIX",
  "amt0": "1000",
  "amt1": "1000",
  "amt0_min": "1000",
  "amt1_min": "1000",
  "doge": 0
}

// remove
{ 
  "p": "pair-v2",
  "op": "remove",
  "pair_id": "CZZ",
  "tick0_id": "CZZ",
  "tick1_id": "UNIX",
  "liquidity": "10", 
  "doge": 0
}

// swap
{ 
  "p": "pair-v2",
  "op": "swap",
  "pair_id":"",
  "tick0_id": "CZZ",
  "amt0": "1000",
  "amt1_min": "1000",
  "doge": 0
}
```

## invite

```json lines lines
// deploy
{ 
  "p": "invite",
  "op": "deploy",
  "invite_address":"Inviter Address"
}
```