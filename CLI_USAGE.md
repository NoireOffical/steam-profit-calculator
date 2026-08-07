# Steam Profit CLI 使用说明

## 1. 工具用途

这个 CLI 用来计算 Steam 商品的挂刀收益，并把结果保存到本地 CSV 文件。

当前支持五个命令：

- `lookup`：查询外部商品标识与市场摘要字段，辅助确认规范商品名和当前市场状态
- `add`：正向计算，一次输入一个商品、一个买价和一个或多个卖价
- `reverse`：反向计算，根据目标收益或目标挂刀比例反推出买价或卖价
- `import`：从 CSV 文件批量导入多个正向计算场景
- `list`：读取并展示已经保存的结果

展示模式：

- 默认使用 `table` 模式，以终端表格形式展示
- 可以通过 `--output csv` 切换为 CSV 风格 stdout 输出
- 落盘存储仍然使用 CSV 文件，不受展示模式影响

核心公式：

- 单件收益：`sell_price * multiplier - buy_price`
- 总收益：`net_profit * quantity`
- 挂刀比例：`buy_price / (sell_price * multiplier)`

默认 `multiplier` 为 `0.837`，也就是沿用之前固定的 Steam 净倍率模型。

## 2. 运行方式

在仓库根目录执行：

```bash
go run ./cmd/steam-profit <command> [flags]
```

例如：

```bash
go run ./cmd/steam-profit lookup --query "AK-47" --limit 5
go run ./cmd/steam-profit add --name "Capsule" --buy 0.48 --sell 0.8
go run ./cmd/steam-profit reverse --name "Capsule" --solve buy --sell 0.8 --target-knife-ratio 0.7
go run ./cmd/steam-profit import --input ./batch.csv
go run ./cmd/steam-profit list --output csv
```

## 3. lookup 命令

### 3.1 基本格式

```bash
go run ./cmd/steam-profit lookup \
  --query "<关键字或商品名>" \
  [--limit <结果数量>] \
  [--output table|csv]
```

规则：

- `--query` 必填
- `--limit` 可选，但如果传入，必须大于 `0`
- `lookup` 只做查询展示，不会写入 `steam-profit-records.csv`

### 3.2 provider 选择与配置

`lookup` 的 provider 通过环境配置选择，不通过 CLI 参数选择。也就是说，切换 provider 时仍然使用同一组命令参数：

```bash
go run ./cmd/steam-profit lookup --query "AK-47" --limit 5 --output table
```

这是一个“接入即用”的适配层：同一查询接口还会被网页、JSON API 和 AI 的只读 `lookup_items` 工具复用。轮换 API Key 或切换 `steam_public` / `steamapis` 不需要改命令参数或迁移记录；外部数据源故障只影响 `lookup`，不会影响 `add`、`reverse`、`import`、`list` 或网站中的计算与历史功能。

当前支持的 provider：

- `steam_public`：Steam Community Market 公开搜索接口，也是默认 provider
- `steamapis`：SteamApis v2 认证关键词搜索；不下载全量目录，也不请求商品详情

如果未设置 `STEAM_PROFIT_LOOKUP_PROVIDER`，默认使用 `steam_public`。如果设置了暂不支持的 provider，命令会直接返回配置错误，不会隐式 fallback 到其他 provider。

可通过以下环境变量覆盖默认配置：

```text
STEAM_PROFIT_LOOKUP_PROVIDER
STEAM_PROFIT_LOOKUP_BASE_URL
STEAM_PROFIT_LOOKUP_APP_ID
STEAM_PROFIT_LOOKUP_TIMEOUT_MS
STEAM_PROFIT_LOOKUP_API_KEY
STEAM_PROFIT_LOOKUP_API_KEY_HEADER
STEAM_PROFIT_LOOKUP_USER_AGENT
```

默认行为：

- `STEAM_PROFIT_LOOKUP_PROVIDER`：默认 `steam_public`
- `STEAM_PROFIT_LOOKUP_BASE_URL`：未设置时跟随 provider；`steam_public` 使用 `https://steamcommunity.com`，`steamapis` 使用 `https://marketplaceapi.steamapis.com`
- `STEAM_PROFIT_LOOKUP_APP_ID`：默认 `730`
- `STEAM_PROFIT_LOOKUP_TIMEOUT_MS`：默认 `5000`
- `STEAM_PROFIT_LOOKUP_API_KEY`：默认空；选择 `steamapis` 时必填
- `STEAM_PROFIT_LOOKUP_API_KEY_HEADER`：默认 `X-API-Key`
- `STEAM_PROFIT_LOOKUP_USER_AGENT`：默认 `steam-profit-calculator/lookup`

例如显式使用默认 provider：

```bash
STEAM_PROFIT_LOOKUP_PROVIDER=steam_public \
go run ./cmd/steam-profit lookup --query "AK-47" --limit 5
```

使用 SteamApis 简单搜索：

```bash
STEAM_PROFIT_LOOKUP_PROVIDER=steamapis \
STEAM_PROFIT_LOOKUP_API_KEY="your-steamapis-key" \
go run ./cmd/steam-profit lookup --query "AK-47 Redline" --limit 5
```

请求行为补充：

- 所有 lookup 请求都会携带 `User-Agent`
- 当 provider 返回 `429 Too Many Requests` 时，客户端会优先遵循 `Retry-After`，否则按短退避自动重试
- 如果多次重试后仍然被限流，命令会显式失败并保留 rate limit 错误信息
- SteamApis 的轻量搜索结果只提供 marketplace 最低美元报价和挂单数；来源会附在 `sell_price_text`，`sale_price_text` 留空
- SteamApis 报价可能来自 Buff163、Skinport 等第三方 marketplace，不能标记为 Steam 官方市场价

### 3.3 输出字段

`lookup` 使用独立字段，不和收益记录字段混用：

```text
item_id,canonical_name,app_name,query_time,sale_price_text,sell_price_text,sell_listings
```

其中 `query_time` 表示这批实时价格快照被查询到的时间，统一使用 UTC `RFC3339` 格式。

例如：

```bash
go run ./cmd/steam-profit lookup --query "AK-47" --limit 2
```

默认 `table` 输出类似：

```text
item_id     canonical_name                      app_name          query_time            sale_price_text  sell_price_text  sell_listings
----------  ----------------------------------  ----------------  --------------------  ---------------  ---------------  -------------
7993054046  AK-47 | Aphrodite (Battle-Scarred)  Counter-Strike 2  2026-07-14T06:15:30Z  $18.45           $19.12                      17
7151910841  AK-47 | Asiimov (Battle-Scarred)    Counter-Strike 2  2026-07-14T06:15:30Z  $10.52           $11.06                      42
```

如果使用 CSV 风格 stdout：

```bash
go run ./cmd/steam-profit lookup --query "AK-47" --limit 1 --output csv
```

```text
item_id,canonical_name,app_name,query_time,sale_price_text,sell_price_text,sell_listings
7993054046,AK-47 | Aphrodite (Battle-Scarred),Counter-Strike 2,2026-07-14T06:15:30Z,$18.45,$19.12,17
```

## 4. add 命令

### 4.1 基本格式

```bash
go run ./cmd/steam-profit add \
  --name "<商品名>" \
  --buy <购入价> \
  [--quantity <数量>] \
  [--multiplier <净倍率>] \
  --sell <出售价>
```

可选显示参数：

```text
--output table|csv
```

### 4.2 quantity 与 multiplier

- `--quantity` 可选，不传时默认按 `1` 计算
- `--multiplier` 可选，不传时默认按 `0.837` 计算
- `total_profit = net_profit * quantity`

例如：

```bash
go run ./cmd/steam-profit add \
  --name "Capsule" \
  --buy 0.48 \
  --quantity 2 \
  --multiplier 0.85 \
  --sell 0.8
```

默认 `table` 输出类似：

```text
item_name  buy_price  sell_price  quantity  net_profit  total_profit  knife_ratio
---------  ---------  ----------  --------  ----------  ------------  -----------
Capsule        0.480       0.800         2       0.200         0.400     0.705882
```

### 4.3 多个 sell 价格

多个出售价要重复写 `--sell`，不要写成逗号拼接字符串。

```bash
go run ./cmd/steam-profit add \
  --name "AK-47 Redline" \
  --buy 100 \
  --quantity 3 \
  --sell 120 \
  --sell 135 \
  --sell 150
```

每个 `sell` 会生成一条记录，并各自落盘。

### 4.4 自定义记录文件

如果不传 `--file`，默认会读写当前目录下的：

```text
steam-profit-records.csv
```

如果你想写到指定文件，可以这样：

```bash
go run ./cmd/steam-profit add \
  --name "Capsule" \
  --buy 0.48 \
  --quantity 5 \
  --sell 0.7 \
  --sell 0.8 \
  --file ./capsule-records.csv
```

## 5. reverse 命令

### 5.1 基本格式

```bash
go run ./cmd/steam-profit reverse \
  --name "<商品名>" \
  --solve buy|sell \
  [--quantity <数量>] \
  [--multiplier <净倍率>] \
  [--buy <已知买价>] \
  [--sell <已知卖价>] \
  [--target-net-profit <目标单件收益>] \
  [--target-total-profit <目标总收益>] \
  [--target-knife-ratio <目标挂刀比例>]
```

规则：

- `--solve sell` 时必须提供 `--buy`
- `--solve buy` 时必须提供 `--sell`
- 三个目标参数里必须且只能提供一个

### 5.2 根据目标总收益反推卖价

```bash
go run ./cmd/steam-profit reverse \
  --name "Capsule" \
  --solve sell \
  --buy 0.48 \
  --quantity 5 \
  --multiplier 0.85 \
  --target-total-profit 1.0
```

示例输出：

```text
item_name  buy_price  sell_price  quantity  net_profit  total_profit  knife_ratio
---------  ---------  ----------  --------  ----------  ------------  -----------
Capsule        0.480       0.800         5       0.200         1.000     0.705882
```

### 5.3 根据目标挂刀比例反推卖价

```bash
go run ./cmd/steam-profit reverse \
  --name "Capsule" \
  --solve sell \
  --buy 0.48 \
  --target-knife-ratio 0.7
```

示例输出：

```text
item_name  buy_price  sell_price  quantity  net_profit  total_profit  knife_ratio
---------  ---------  ----------  --------  ----------  ------------  -----------
Capsule        0.480       0.819         1       0.206         0.206     0.700000
```

### 5.4 根据目标挂刀比例反推买价

```bash
go run ./cmd/steam-profit reverse \
  --name "Capsule" \
  --solve buy \
  --sell 0.8 \
  --target-knife-ratio 0.7
```

## 6. import 命令

### 6.1 基本格式

```bash
go run ./cmd/steam-profit import --input <场景文件路径> [--file <记录文件>] [--output table|csv]
```

`--input` 是必填参数。

### 6.2 导入文件格式

批量导入文件必须使用以下表头：

```text
item_name,buy_price,sell_price,quantity,multiplier
```

说明：

- 每一行代表一个正向计算场景
- `quantity` 可留空，留空时默认按 `1`
- `multiplier` 可留空，留空时默认按 `0.837`
- 当前版本不支持在一个单元格里写多个卖价

示例：

```csv
item_name,buy_price,sell_price,quantity,multiplier
Capsule,0.48,0.7,5,
Capsule,0.48,0.8,5,0.85
```

命令示例：

```bash
go run ./cmd/steam-profit import --input ./batch.csv --output csv
```

示例输出：

```text
item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio
Capsule,0.480,0.700,5,0.106,0.529,0.819252
Capsule,0.480,0.800,5,0.200,1.000,0.705882
```

原子性说明：

- 导入前会先校验整个文件
- 只要有任意一行非法，本次导入就不会部分落盘

## 7. list 命令

### 7.1 基本格式

```bash
go run ./cmd/steam-profit list [--file <记录文件>] [--output table|csv]
```

它默认读取：

```text
./steam-profit-records.csv
```

如果文件还是旧版 5 列格式，程序会在读取时兼容：

- `quantity` 按 `1` 处理
- `total_profit` 按 `net_profit` 处理

### 7.2 CSV 风格显示

```bash
go run ./cmd/steam-profit list --output csv --file ./capsule-records.csv
```

## 8. 字段说明

当前要区分两套字段：

- 终端展示 / `--output csv` 输出字段
- 落盘存储 CSV 字段

### 8.1 展示字段

终端表格输出和 `--output csv` 使用以下 7 列：

```text
item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio
```

字段含义：

- `item_name`：商品名
- `buy_price`：购入价
- `sell_price`：出售价
- `quantity`：数量
- `net_profit`：单件收益
- `total_profit`：总收益
- `knife_ratio`：挂刀比例

### 8.2 存储字段

落盘到记录文件的 CSV 使用以下 8 列：

```text
item_name,buy_price,sell_price,quantity,net_profit,total_profit,knife_ratio,fee
```

其中前 7 列与展示字段一致，新增：

- `fee`：手续费/倍率标记
  - 如果本次计算使用默认 multiplier，则写 `default`
  - 如果本次计算显式指定了 multiplier，则写对应数值，例如 `0.850`

格式规则：

- 价格保留 3 位小数
- 挂刀比例保留 6 位小数
- 显式 multiplier 落盘时保留 3 位小数

## 9. 常见错误

### 9.1 没带命令

```bash
go run ./cmd/steam-profit
```

输出：

```text
usage: steam-profit <command> [flags]
commands: lookup, add, reverse, import, list
```

### 9.2 lookup 缺少 query

```bash
go run ./cmd/steam-profit lookup
```

输出：

```text
steam-profit lookup: --query is required
```

### 9.3 lookup 无匹配结果

```bash
go run ./cmd/steam-profit lookup --query "definitely-not-found-item"
```

输出类似：

```text
lookup items: no matching items found
```

### 9.4 缺少 sell

```bash
go run ./cmd/steam-profit add --name "AK-47 Redline" --buy 100
```

输出：

```text
at least one sell price is required
```

### 9.5 非法 quantity

```bash
go run ./cmd/steam-profit add --name "Capsule" --buy 0.48 --quantity 0 --sell 0.8
```

输出：

```text
quantity must be greater than 0
```

### 9.6 reverse 目标参数冲突

```bash
go run ./cmd/steam-profit reverse \
  --name "Capsule" \
  --solve buy \
  --sell 0.8 \
  --target-net-profit 0.19 \
  --target-knife-ratio 0.7
```

输出：

```text
exactly one reverse target is required
```

### 9.7 import 文件有非法行

```bash
go run ./cmd/steam-profit import --input ./bad-batch.csv
```

这类错误会指出具体出错的导入行，例如：

```text
import records: invalid sell price on import row 3: strconv.ParseFloat: parsing "abc": invalid syntax
```

## 10. 常用示例

### 示例 1：先查询商品名

```bash
go run ./cmd/steam-profit lookup --query "AK-47" --limit 3
```

### 示例 2：测试单个价格

```bash
go run ./cmd/steam-profit add --name "test-item" --buy 0.48 --sell 0.8
```

### 示例 3：显式指定 multiplier

```bash
go run ./cmd/steam-profit add \
  --name "test-item" \
  --buy 0.48 \
  --quantity 2 \
  --multiplier 0.85 \
  --sell 0.8
```

### 示例 4：根据目标总收益反推卖价

```bash
go run ./cmd/steam-profit reverse \
  --name "test-item" \
  --solve sell \
  --buy 0.48 \
  --quantity 5 \
  --target-total-profit 1.0
```

### 示例 5：批量导入并输出 CSV 风格 stdout

```bash
go run ./cmd/steam-profit import --input ./batch.csv --output csv
```

### 示例 6：读取自定义文件

```bash
go run ./cmd/steam-profit list --file ./capsule-records.csv
```
