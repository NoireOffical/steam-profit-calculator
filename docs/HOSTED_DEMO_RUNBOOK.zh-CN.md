# 受控托管演示运行手册

本手册定义 `v0.6.0` 受控在线演示的最低运行契约。当前仓库已经实现平台无关的访问控制、限流、模型配额、只读线上检查和手动真实模型评测，并已在 Railway 完成单副本、HTTPS、Secret、持久卷、重启持久化和手动恢复验收。受控地址是 [steam-profit-calculator-production.up.railway.app](https://steam-profit-calculator-production.up.railway.app)；健康检查公开，交互页面需要单独提供的评审凭据。

这套边界用于“把作品集安全地给少量评审访问”，不是完整账号系统，也不代表项目已经适合公开运营。

## 最低托管契约

托管环境必须同时满足以下条件：

- 只运行一个应用副本；当前 SQLite 不支持多个实例共享写入；
- 公网入口必须使用 HTTPS，HTTP 只能用于本机冒烟；
- 使用仓库 `Dockerfile` 构建并以非 root 运行；应用只把持久数据写入 `/data`，平台支持时再启用只读根文件系统；
- 把可持久化卷挂载到 `/data`，数据库固定为 `/data/steam-profit.db`；
- 所有凭据通过平台 Secret 或环境变量注入，不写入镜像、部署清单、日志或 Git；
- 健康检查访问公开的 `GET /api/health`；其他页面与 API 由演示凭据保护；
- 转发头只有在平台明确提供客户端地址头时才可被信任；Railway 直接入口使用其 `X-Real-IP`；
- 停止应用时允许至少 10 秒优雅退出；回滚必须使用固定 Tag 对应的不可变镜像。

推荐的生产演示环境值如下；实际密码不要直接写进命令历史：

| 变量 | 推荐值 | 说明 |
| --- | --- | --- |
| `PORT` | Railway 自动注入 | 未显式设置服务地址时，应用监听 `0.0.0.0:$PORT` |
| `STEAM_PROFIT_STORAGE` | `sqlite` | 当前受支持的单实例仓储 |
| `STEAM_PROFIT_DATABASE_FILE` | `/data/steam-profit.db` | 必须位于持久卷 |
| `STEAM_PROFIT_DEMO_USERNAME` | 平台 Secret | 演示评审用户名；不能包含冒号或换行 |
| `STEAM_PROFIT_DEMO_PASSWORD` | 平台 Secret | 至少 12 个字符；可随时轮换 |
| `STEAM_PROFIT_RATE_LIMIT_PER_MINUTE` | `120` | 每客户端、每分钟 API 请求上限；`0` 表示关闭 |
| `STEAM_PROFIT_AI_RUNS_PER_HOUR` | `12` | 整个实例每小时实际启动的模型调用上限；`0` 表示关闭 |
| `STEAM_PROFIT_TRUST_PROXY_HEADERS` | Railway 直连时设 `true` | 读取 Railway 提供的 `X-Real-IP`；更换代理链时重新评估 |

`GET /api/health` 保持公开，便于平台判断数据库依赖是否可用。浏览器首页、静态资源和其余 API 在配置了用户名与密码后都需要 HTTP Basic 凭据。该方案只有一组共享演示凭据，不提供注册、找回密码、角色、租户或用户数据隔离。

## Railway 部署配置

仓库根目录的 `railway.toml` 固定使用现有 `Dockerfile`，把 `/api/health` 设为部署健康检查，失败时最多重启 3 次，并给应用 10 秒排空时间。服务端优先读取显式的 `STEAM_PROFIT_SERVER_ADDR`；未设置时会读取 Railway 注入的 `PORT` 并监听所有容器网卡。非法端口会在打开数据库前阻止启动。

Railway 项目必须按以下顺序配置：

1. 从 GitHub 仓库 `NoireOffical/steam-profit-calculator` 的 `main` 分支创建一个服务；
2. 确认构建来源显示根目录 `Dockerfile`，部署配置来自 `/railway.toml`；
3. 只保留 1 个副本，并选择离主要评审较近的单一区域；
4. 给同一服务添加一个 Railway Volume，挂载路径必须是 `/data`；Railway 不接受 Dockerfile 的 `VOLUME` 指令，因此持久化只由平台挂载配置负责；
5. 生成 Railway 公共域名；Railway 入口负责 HTTPS，应用容器内部继续使用 HTTP；
6. 在服务 Variables 中注入下表变量，不要把真实值写回仓库；
7. 首次部署成功后运行只读 hosted smoke，再做计算、重启和历史读取验收。

Railway 服务变量：

| 变量 | Railway 值 |
| --- | --- |
| `STEAM_PROFIT_STORAGE` | `sqlite` |
| `STEAM_PROFIT_DATABASE_FILE` | `/data/steam-profit.db` |
| `STEAM_PROFIT_IMPORT_CSV` | `/app/demo/seed.csv` |
| `STEAM_PROFIT_DEMO_USERNAME` | 随机或专用评审用户名 |
| `STEAM_PROFIT_DEMO_PASSWORD` | 至少 12 位的随机 Secret |
| `STEAM_PROFIT_RATE_LIMIT_PER_MINUTE` | `120` |
| `STEAM_PROFIT_AI_RUNS_PER_HOUR` | `12` |
| `STEAM_PROFIT_TRUST_PROXY_HEADERS` | `true` |
| `STEAM_PROFIT_LOOKUP_PROVIDER` | 默认 `steam_public`；准备好 SteamApis Secret 后可切换为 `steamapis` |
| `STEAM_PROFIT_LOOKUP_API_KEY` | 仅选择 `steamapis` 时设置为 Railway Secret |
| `STEAM_PROFIT_AI_PROVIDER` | 首次部署保持 `disabled`；模型验收后再改为 `openai` |
| `STEAM_PROFIT_AI_API_KEY` | 仅启用 `openai` 时设置为 Railway Secret；也可使用 `OPENAI_API_KEY` |
| `RAILWAY_RUN_UID` | `0`；仅允许容器入口修正 root 所有的卷挂载点，随后立即降权为 `steam-profit` 用户运行应用 |

不要设置 `STEAM_PROFIT_SERVER_ADDR`，让应用跟随 Railway 的 `PORT`。Railway Volume 以 root 所有者挂载；官方建议非 root 镜像配置 `RAILWAY_RUN_UID=0`。本镜像只用该权限修正 `/data` 顶层目录的所有者，然后通过 `su-exec` 立即以 `steam-profit` 用户启动服务器，默认的 Compose 路径仍直接以非 root 用户运行。参见 [Using Volumes — Permissions](https://docs.railway.com/volumes#permissions)。Railway 官方说明其公共入口使用 HTTPS，并提供 `X-Real-IP` 表示客户端地址：[Public networking specs and limits](https://docs.railway.com/networking/public-networking/specs-and-limits)。Railway 带 Volume 的服务在更新时不会同时把同一个卷挂载到两个部署，因此发布切换会有短暂停机：[Healthchecks — services with attached volumes](https://docs.railway.com/deployments/healthchecks#services-with-attached-volumes)。这与当前单实例 SQLite 边界一致。

### `v0.6.0` 实际部署基线

截至 2026-08-07，当前环境使用 Railway Trial、EU West 单副本、挂载到 `/data` 的 0.5 GB Volume 和 Railway 托管 HTTPS。数据库固定为 `/data/steam-profit.db`，部署状态为 Online；共享用户名和随机密码只保存在 Railway Variables 与 GitHub Actions Secrets 中，不写入仓库。线上 `STEAM_PROFIT_AI_PROVIDER=disabled`，因此演示不会产生模型费用，也不能展示真实模型回答；AI 是可选适配器，这一状态不影响查询、计算、反推和 SQLite 历史工作台。

首次部署实际暴露了 Railway Volume 的 root 所有权边界。容器现在保持默认 `USER steam-profit`；只有 Railway 显式设置 `RAILWAY_RUN_UID=0` 时，入口脚本才修正 `/data` 顶层目录并立即降权。该修复已通过线上健康检查和重启持久化验收。

## 模型密钥与手动质量评测

如果线上演示启用 OpenAI Provider，把 `OPENAI_API_KEY` 或 `STEAM_PROFIT_AI_API_KEY` 保存为托管平台 Secret。不要把密钥写进 `.env.example`、工作流文件、镜像层或公开仓库。OpenAI 官方也建议通过环境变量或密钥管理服务加载密钥，并避免在代码和公开仓库中暴露密钥：[Production best practices — API keys](https://developers.openai.com/api/docs/guides/production-best-practices#api-keys)。

GitHub 仓库可配置：

| 类型 | 名称 | 用途 |
| --- | --- | --- |
| Actions Secret | `OPENAI_API_KEY` | 仅供手动 `Live AI Evaluation` 工作流使用 |
| Actions Variable | `OPENAI_MODEL` | 可选；覆盖评测模型 |
| Actions Secret | `STEAM_PROFIT_DEMO_USERNAME` | 线上只读检查的演示用户名 |
| Actions Secret | `STEAM_PROFIT_DEMO_PASSWORD` | 线上只读检查的演示密码 |
| Actions Variable | `STEAM_PROFIT_DEMO_URL` | HTTPS 演示根地址 |

`Live AI Evaluation` 只能手动触发。默认分别对当前 Prompt 和基线 Prompt 执行 3 条版本化语义用例，并上传不含 Prompt、模型原文、工具值和密钥的 JSON 报告。调用费率与成本上限由触发者在当次运行中填写，普通 CI 不会产生模型费用。

轮换模型密钥时，先在供应商创建新密钥并替换平台 Secret，重启单实例验证助手状态，再撤销旧密钥。演示密码也可以直接替换后重启；替换后同步更新 GitHub Actions Secret。

## 本地受保护模式冒烟

先构建服务，再通过你自己的安全环境注入凭据：

```bash
make build
export STEAM_PROFIT_DEMO_USERNAME="reviewer"
export STEAM_PROFIT_DEMO_PASSWORD="replace-with-a-long-local-password"
export STEAM_PROFIT_RATE_LIMIT_PER_MINUTE="120"
export STEAM_PROFIT_AI_RUNS_PER_HOUR="12"
./.build/steam-profit-server --addr 127.0.0.1:8080 --db .build/hosted-demo.db
```

另一个终端运行只读检查。只有本机 HTTP 测试才允许显式绕过 HTTPS 要求：

```bash
export STEAM_PROFIT_DEMO_URL="http://127.0.0.1:8080"
export STEAM_PROFIT_ALLOW_HTTP_SMOKE="true"
bash script/hosted_smoke.sh
```

脚本依次确认公开健康检查、匿名首页 `401`、认证后的浏览器页面和助手状态。它不会提交计算、写入记录，也不会发起付费模型请求。

## 线上检查与最小告警

`Hosted Demo Smoke` 工作流每 6 小时运行一次，也可手动提供临时 URL。配置 `STEAM_PROFIT_DEMO_URL` 后，它执行与本地相同的只读检查；失败时 GitHub Actions 会把本次工作流标记为失败，作为当前的最小告警渠道。

实际托管环境的重启持久化验收使用手动 `Hosted Persistence Exercise` 工作流。先以 `prepare` 模式写入唯一的版本化验收记录，再从 Railway 重启或重新部署同一服务，最后以 `verify` 模式确认该记录仍存在。该工作流不会定时运行，避免日常巡检产生写入副作用。

`v0.6.0` 已保留以下 GitHub Actions 证据：

- [Hosted Demo Smoke：健康、认证边界、浏览器标记和助手关闭状态](https://github.com/NoireOffical/steam-profit-calculator/actions/runs/31154963602)；
- [Hosted Persistence Exercise：写入唯一版本化标记](https://github.com/NoireOffical/steam-profit-calculator/actions/runs/31155723364)；
- [Hosted Persistence Exercise：Railway 重启后读取同一标记](https://github.com/NoireOffical/steam-profit-calculator/actions/runs/31155897200)。

当前日志和指标仍是进程内数据：

- 日志只包含请求方法、路径、状态、耗时和随机请求 ID；
- 指标只累计请求、助手调用、未授权、限流和模型配额拒绝次数；
- 重启会清空指标；仓库尚未接入远程采集、长期趋势、通知升级或值班系统。

## 备份、恢复与回滚

### 备份

至少在每个发布和数据库迁移前生成一次一致性备份。平台支持卷快照时优先使用卷快照；否则先冻结外部写入，再归档整个 `/data` 目录。能够离线访问卷时可以停止实例；只能通过运行中服务下载文件时，必须选择无写入窗口并在导出后验证 SQLite 完整性。每份备份都要记录对应的应用 Tag。

不要把数据库备份提交到 Git。备份可能包含演示计算记录，应保存到受控对象存储或平台备份空间，并设置保留期。

当前 Railway Trial 页面不提供托管 Backups/PITR；启用该能力需要升级 Pro，本里程碑没有执行任何付费升级。因此当前可用路径是 Railway CLI 的完整服务文件导出。Railway CLI 需要连接运行中的服务，正式备份时应先阻止评审写入、等待在途请求结束，再下载整个 `/data`，不能只复制主数据库文件而遗漏 `-wal` 与 `-shm`：

```bash
railway link
railway service files download /data ./railway-data-backup
sqlite3 ./railway-data-backup/steam-profit.db "PRAGMA integrity_check;"
```

导出目录和其中的凭据无关，但可能包含演示历史数据，必须按数据库备份处理。CLI 所需 SSH 公钥应使用临时、唯一命名的密钥，完成下载后同时从 Railway 和本机移除。

### 恢复演练

恢复时先创建新卷或临时服务，不覆盖当前卷：

1. 把备份恢复到新卷的 `/data/steam-profit.db`；
2. 使用同一固定 Tag 启动一个单副本实例；
3. 检查 `/api/health`，再运行 `script/hosted_smoke.sh`；
4. 人工确认历史记录数量和最近一条记录；
5. 验收后再切换流量，保留旧卷直到回滚窗口结束。

2026-08-07 的实际演练从 Railway 下载了完整 `/data`（主文件及 WAL sidecar），`PRAGMA integrity_check` 返回 `ok`。副本随后由当前服务器二进制在独立本机端口启动，公开健康检查返回成功，并读取到平台重启前写入的唯一 `Railway Hosted Persistence v0.6` 标记。演练结束后，临时服务已优雅停止，Railway 临时 SSH 公钥、本机私钥和本地数据库副本均已删除；原始 Railway Volume 未修改。

这证明当前数据可以被导出并由同版本应用恢复读取，但不等于托管快照、PITR、异地长期保留或新 Railway Volume 的平台级恢复已经可用。升级套餐前，每次发布都需要重新执行手动导出，并把加密备份保存到受控位置。

### 应用回滚

当前稳定回滚点是 `v0.5.0`。回滚时固定旧 Tag 的镜像，保留同一持久卷并保持单实例；先检查该版本是否理解当前数据库迁移，再切换流量。若存储兼容性存在疑问，使用发布前备份恢复到新卷，不要降级覆盖唯一数据副本。

更具体的本地 SQLite 和 Compose 归档命令见[演示与运行手册](DEMO_RUNBOOK.zh-CN.md)。

## 当前边界与剩余升级项

`v0.6.0` 已达到“少量评审可访问的受控在线演示”：Railway 单副本、HTTPS、Secret、`/data` Volume、公开依赖健康检查、认证页面、定时只读 smoke、重启持久化和手动导出/隔离恢复均有实际证据。

它仍不能称为公开运营服务：

1. 当前只有共享 Basic 凭据，没有注册、角色、租户或用户数据隔离；
2. 限流、模型配额和指标存在于单进程内，重启会清空且无法跨副本协调；
3. Railway Trial 没有托管 Backups/PITR，当前恢复证据是手动全卷导出后的隔离恢复；
4. 线上 OpenAI 适配器保持关闭，没有临时模型密钥，也没有付费当前/基线真实模型报告；确定性网站功能不受影响，但仍不能宣称真实模型质量已经线上验收；
5. SQLite 只支持当前单副本边界，多实例和正式用户场景需要 PostgreSQL 与在线迁移。
