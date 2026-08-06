# Steam Profit Lab：3–5 分钟演示与运行手册

本手册用于复现“浏览器 → HTTP API → 确定性计算 → SQLite → 可观测状态”的完整路径。默认不启用真实 Steam 查询或 AI Provider，不需要任何密钥。

## 路径一：原生本地演示

环境要求：Go 1.21+、Node.js、pnpm。首次运行浏览器测试时还需要网络下载 Chromium。

```bash
make demo-run
```

打开 `http://127.0.0.1:8080`。演示数据库位于 `.build/demo/steam-profit.db`，启动时会通过正式 CSV 导入器加载 12 条 `Demo Capsule` 记录；重复启动不会重复导入。

### 建议演示顺序

1. 确认右上角显示“计算引擎已连接”，助手区域显示“AI 策略助手未启用”，同时收益表单仍可使用。
2. 在历史区筛选 `Demo Capsule`，每页选择 10 条，展示第一页、下一页和仅亏损筛选。
3. 新建一个收益场景，例如买入价 `0.480`、数量 `2`、卖价 `0.450, 0.800`；展示服务端保存后的记录 ID、时间和最佳总收益。
4. 刷新浏览器并再次筛选刚才的商品，证明历史来自 SQLite，而不是浏览器缓存。
5. 查看健康和指标：

```bash
curl --fail http://127.0.0.1:8080/api/health
curl --fail http://127.0.0.1:8080/api/v1/metrics
```

指标只包含进程启动时间、请求计数、响应分类、耗时以及聚合的助手 Token/工具次数，不包含商品、查询参数、Prompt、模型回答、密钥或数据库内容。

## 路径二：Docker Compose 演示

环境要求：Docker Engine 或 Docker Desktop，并支持 `docker compose`。

```bash
docker compose up --build --wait
```

打开 `http://127.0.0.1:8080`。Compose 使用固定命名卷 `steam-profit-demo-data` 保存 `/data/steam-profit.db`，容器重建不会清空历史，种子数据也不会重复导入。

```bash
docker compose ps
docker compose logs --follow app
docker compose stop
docker compose start --wait
docker compose up --force-recreate --wait
docker compose down
```

`docker compose down` 默认保留命名卷。只有明确执行 `docker compose down --volumes` 才会删除演示数据。

## 浏览器自动验收

```bash
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
make e2e
```

四条 Chromium 用例会在临时数据库中验证：

- 计算保存后刷新页面仍可读取；
- 12 条固定种子记录的筛选与两页分页；
- 浏览器输入错误不会生成历史；
- AI 关闭时确定性计算仍然可用。

测试结束后临时数据库会被清理，不会写入正式的 `steam-profit.db`。

## SQLite 备份与恢复

### 原生数据库

先停止服务，再使用 SQLite 的在线备份命令生成独立文件：

```bash
mkdir -p backups
sqlite3 .build/demo/steam-profit.db ".backup 'backups/steam-profit-demo.db'"
sqlite3 backups/steam-profit-demo.db "PRAGMA integrity_check;"
```

恢复时不要覆盖正在使用的数据库。先把备份作为新的数据库启动并核对历史：

```bash
./.build/steam-profit-server \
  --addr 127.0.0.1:8081 \
  --db ./backups/steam-profit-demo.db
```

### Compose 命名卷

先停止应用，再把整个 `/data` 目录归档：

```bash
docker compose stop
mkdir -p backups
docker run --rm \
  --volume steam-profit-demo-data:/data:ro \
  --volume "$PWD/backups:/backup" \
  alpine:3.24.1 \
  tar -czf /backup/steam-profit-demo-data.tar.gz -C /data .
docker compose start --wait
```

恢复演练使用一个新卷，避免覆盖原数据：

```bash
docker volume create steam-profit-demo-restore
docker run --rm \
  --volume steam-profit-demo-restore:/data \
  --volume "$PWD/backups:/backup:ro" \
  alpine:3.24.1 \
  tar -xzf /backup/steam-profit-demo-data.tar.gz -C /data
STEAM_PROFIT_DATA_VOLUME=steam-profit-demo-restore \
STEAM_PROFIT_PORT=8081 \
docker compose up --build --wait
```

确认恢复结果后停止该演练实例。删除卷属于不可恢复操作，只有在确认不再需要数据时才执行。

## 回滚与边界

- 应用回滚：停止当前服务，用上一个固定 Tag 的二进制读取同一 SQLite 文件；已经发布的迁移不会被修改。
- 存储回滚：使用 `--storage=csv --file <path>` 启动旧版 CSV 模式。
- Compose 回滚：固定旧镜像版本并保留同一命名卷；单实例 SQLite 不支持横向扩容。
- 当前日志和指标仅存在于运行进程，没有远程采集、告警或长期保留。
- 当前仍无登录、租户隔离、限流、托管备份或公网 TLS，不应直接作为公开运营服务。
