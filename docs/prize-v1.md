# prize-v1：通用奖池

奖池是一个由协议持有的 DRC-20 储备：

- 任何人可以创建奖池、向奖池存入。
- 指定的结算方公布各地址的**累计**可领额。
- 用户自己领取「累计 − 已领」。
- 结算方长期不处理存入时，存入人可以自己退款。

indexer 不知道奖池用来做什么（大逃杀等玩法都在 indexer 之外），只执行下面的规则。

来源：dogeuni-trade `docs/PRIZE_POOL_OP.md` §4（c0591eb），以及 indexer 侧评审改动（见文末）。

## 激活

- `explorer.prize_v1_height`：从这个高度起才分发 `p:"prize-v1"`。0 表示不启用。
- 激活前的 prize-v1 铭文按无效处理，不记录。
- 所有 indexer 实例必须用同一版本代码和同一高度，否则账本分叉。

## 身份

- 每个 op 的发起人 `holder` = reveal 的 `vout[0]` 地址，并且必须等于 `commitSender`（commit `vin[0]` 花费的输出的地址），和 stake-v2 相同。
- 储备地址不能当发起人（`reserve_sender_height`，PR #13）。

## 储备地址

```
reserve = P2SH( OP_RETURN <push "prize-v1:" + pool_id> )
```

赎回脚本以 `OP_RETURN` 开头，执行即失败，所以谁都花不掉。不沿用 `P2SH(字符串)`：那个字符串本身就是赎回脚本，某些字符串执行结果为真，任何人都能花（PR #13）。

## 铭文

金额、高度、seq 都是十进制整数字符串，单位是 tick 的最小单位。

| op | 谁能发 | 字段 |
|---|---|---|
| `create` | 任何人，成为 `creator` | `tick`, `settler`, `timeout` |
| `deposit` | 任何人 | `pool_id`, `amt`, `memo`（可选） |
| `settle` | 当前 `settler` | `pool_id`, `seq`, `upto`, `entries: [[地址, 累计额], …]` |
| `claim` | 任何人（领自己的） | `pool_id` |
| `refund` | 存入人 | `pool_id` |
| `withdraw` | `creator` | `pool_id` |
| `config` | `creator` | `pool_id`, `settler` |
| `close` | `creator` | `pool_id` |

- `pool_id` = create 交易的 txid。
- 一笔铭文的 JSON 受共识单个 push ≤ 520 字节限制，一笔 settle 约 5 个地址；地址多就分几笔，`seq` 递增。

## 状态

奖池 `prize_pool`：

- `tick`, `creator`, `settler`, `timeout`, `reserves_address`
- `seq`：初始 0
- `upto`：初始为 create 高度 − 1，所以同一块里 create 之后的存入算未处理
- `last_settle_height`：初始为 create 高度
- `owed`：未付总额 Σ(cum − claimed)
- `status`：open 或 closed

其它表：

- 每个地址的可领额 `prize_entitlement`：`cum`、`claimed`。
- 存入 `prize_deposit`：`holder`、`amt`、`memo`、`block_number`、`refunded`。

派生量（按当前状态计算）：

- `balance` = 储备地址该 tick 的 DRC-20 余额。直接用普通 transfer 打进储备的钱也算在内，可以被分配，但不算存入、不能退款。
- `pending` = Σ 未退款且 `block_number > upto` 的存入。
- `expired(h)` = 存在未处理存入，且 `h > 最早一笔未处理存入的高度 + timeout`。没有未处理存入时永不过期。
- `closed(h)` = `status = closed` 或 `expired(h)`。

**不变式**：每个 op 之后都有 `owed + pending ≤ balance`。

- deposit、claim、refund、withdraw 都是在 `balance` 和 `owed` / `pending` 两边加减同样的数（withdraw 正好取走盈余），不可能破坏不变式。
- 只有 settle 需要显式检查。

## 校验与效果

任何一条校验不过，整笔失败：`order_status = 1` 加 `err_info`，不动钱，也不改任何状态。每笔 op 的全部写入和 `order_status = 0` 放在同一个数据库事务里。

**create**

- `tick` 是已部署的 DRC-20。
- `settler` 是合法的 Doge 主网地址：`btcutil.DecodeAddress` 能解析，并且 `EncodeAddress()` 和原串相同。
- `timeout` ∈ [1440, 525600]。
- 效果：建 `prize_pool`。

**deposit**

- 奖池存在，且 `!closed(h)`。
- `amt` > 0。
- `memo`：可选，≤ 64 字节，只允许可打印 ASCII（0x20–0x7e）。生产是 MySQL，限 ASCII 是为了避开 utf8 / utf8mb4 的差异。
- 效果：`TransferDrc20(holder → reserve, amt)`，并记一行存入。

**settle**

- `holder = settler`，且 `!closed(h)`。
- `seq = pool.seq + 1`。同一块里交易的先后由矿工决定，结算方必须串行发：下一笔 commit 花上一笔的找零。
- `pool.upto ≤ upto < h`。
- `entries`：
  - 非空，≤ 50 条；
  - 每个地址合法（同 create 的规则），不能是本池储备，同一笔里不能重复；
  - 累计额是 ≥ 0 的整数，并且不小于该地址当前的 `cum`。已公布的额度收不回。
- 写入后检查 `owed' + pending(upto') ≤ balance`。
- 效果：更新 entitlement 的 `cum`、`owed`、`seq`、`upto`、`last_settle_height = h`。
- 同一份累计额重发没有额外效果（cum 不变，owed 不变），所以后端重发不会双付。

**claim**

- 该地址 `cum − claimed > 0`。池子关闭后也能领。
- 效果：`TransferDrc20(reserve → holder, cum − claimed)`；`claimed = cum`；`owed` 减去同样的数。

**refund**

- `closed(h)`。
- `holder` 至少有一笔未退款、且 `block_number > upto` 的存入。
- 效果：
  - 把这些存入的总额 `TransferDrc20(reserve → holder)`，逐笔标记 `refunded`；
  - 如果 `status` 还是 open，就置为 closed。此后奖池不再过期，也不再收存入和结算。

**withdraw**

- `holder = creator`，`closed(h)`。
- `balance − owed − pending > 0`。
- 效果：把这个差额 `TransferDrc20(reserve → creator)`。

**config**

- `holder = creator`，`!closed(h)`，新 `settler` 合法。
- 效果：更换 `settler`。

**close**

- `holder = creator`，`!closed(h)`。
- 效果：`status = closed`。已公布的额度照样能领，未处理的存入可以退，剩余由 creator 用 withdraw 取走。

## 回滚

- 操作记录（`prize_info`）、存入、每笔 settle 的条目都带 `block_number`。
- fork 到高度 h：删除这几张表里 `block_number > h` 的行，然后对受影响的奖池用剩下的成功操作（`order_status = 0`）重算 `prize_pool` 和 `prize_entitlement`。
- 储备的 DRC-20 余额由 `TransferDrc20` 写的 `Drc20Revert` 照常回滚。

## 评审改动（相对 dogeuni-trade §4）

1. 储备改用 `OP_RETURN` 赎回脚本，并把 `prize_pool` 加入储备表清单，防止储备地址冒充发起人。
2. 超时从「最早一笔未处理存入」起算，而不是 `last_settle_height`，否则结算方只要重发同样的累计额，就能让存入人永远退不了款。过期在每个 op 里按高度判断；新增 creator 的 `close`。
3. 结算方必须串行发 settle（见 settle 里的 `seq` 一条）。
4. 回滚改为删除高度 h 之后的行，再按剩下的成功操作重算。
5. 地址必须是规范写法，memo 只允许 ASCII，`entries` 有上限。
