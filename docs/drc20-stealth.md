# drc-20 隐私转账：索引器接口

隐私转账用一次性地址收款。链上看起来就是普通 transfer，没有任何标记，也不改变记账规则。

## 钱包派生约定

- 发送方公钥 `A`：签名 commit 交易 `Vin[0]` 的公钥，也就是 token 持有地址的公钥。
- outpoint：commit 交易 `Vin[0]` 花费的 `txid:vout`。
- 收款方公布 `B_scan`、`B_spend`。一次性地址由 `A` 和 `B_scan` 的 ECDH 结果派生，并混入 outpoint。不混入的话，同一个 `A` 给同一收款方转两次会得到同一个地址。
- 具体派生算法由钱包定义，索引器只提供 `A` 和 outpoint。

## 接口

### `POST /v4/drc20/transfer-scan`

按索引顺序返回已成功的 transfer，供钱包逐笔试算。

请求：`{"after_id": 0, "from_block": 0, "limit": 100}`，`limit` 上限 1000。

返回 `data` 每行：`id`、`tx_hash`、`block_number`、`sender_pubkey`、`sender_txid`、`sender_vout`、`tick`、`amt`、`to_addresses`。下一页把最后一行的 `id` 作为 `after_id`。

### `POST /v4/drc20/balances`

批量查非零余额。

请求：`{"addresses": ["D..."], "tick": ""}`，地址 1 到 100 个，`tick` 可省略。

返回 `data` 每行：`tick`、`holder_address`、`amt`。

## 约束

- **收款方数量：** 多收款方时每个地址记同样的 `amt`，无法表达“付 30、留 70”，钱包按整笔转出设计。
- **只覆盖部署之后：** `drc20_transfer_key` 不回填历史，从部署后的区块开始才有数据。
- **只支持压缩公钥 P2PKH：** commit `Vin[0]` 不是这种输入时不记录，这笔 transfer 不会出现在 scan 里，但记账不受影响。
- **reorg：** 被回滚的记录会删除，重新打包后拿到新 `id`。钱包已读过的记录可能失效，余额以 `/v4/drc20/balances` 为准，并等足够确认。
- **查询隐私：** 服务端能看到一次请求里的全部地址，等于知道它们同属一个钱包。要保持地址不关联，用户必须查自己的索引器。
- **持币人数：** 每个一次性地址都算一个 holder，链上没有标记，索引器无法区分，所以这个数字会偏高。
