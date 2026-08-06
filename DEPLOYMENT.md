# 部署指南

本文档是本项目构建、发布、生产部署和运行维护的唯一事实来源。构建命令、版本注入方式、二进制路径、systemd 单元、监听地址、端口、公开域名、健康检查、日志或回滚方式发生变化时，必须在同一任务和同一提交中更新本文档。

## 当前生产架构

| 项目 | 当前值 |
| --- | --- |
| 源码目录 | `/root/skills/skillshare` |
| Git 远端 | `origin`: `git@github.com:Chengche123/skillshare.git`；`upstream`: `https://github.com/runkids/skillshare.git` |
| 构建产物 | `/root/skills/skillshare/bin/skillshare` |
| 正式二进制 | `/usr/local/bin/skillshare` |
| 进程管理 | systemd 系统单元 `skillshare-ui.service` |
| 运行用户 | `root`，`HOME=/root` |
| 启动模式 | 全局 Web UI，前台进程由 systemd 托管 |
| 监听地址 | `127.0.0.1:19420`，不直接监听公网地址 |
| 本地首页 | `http://127.0.0.1:19420/` |
| 健康检查 | `GET http://127.0.0.1:19420/api/health` |
| 公开入口 | `https://skill.rainite.com`，由宿主机现有 Cloudflare Tunnel 暴露 |
| 日志 | systemd journal，单元名 `skillshare-ui` |
| 重启策略 | `Restart=on-failure`，间隔 3 秒 |
| 回滚 | `/usr/local/bin/skillshare.backup-<UTC时间>` |

正式实例只使用 systemd 管理。不要使用 `screen`、`nohup`、后台 `skillshare ui start`、裸 Docker 容器或第二个进程管理器托管同一端口。

## 责任边界

本项目负责：

- 构建带内嵌 React UI 的单文件 Go 二进制。
- 安装 `/usr/local/bin/skillshare`。
- 管理 `/etc/systemd/system/skillshare-ui.service`。
- 验证本地监听、首页、健康接口和 journal 日志。

以下内容由宿主机外部配置管理，不属于本仓库的部署流程：

- Cloudflare Tunnel 服务、Tunnel Token 和凭据文件。
- `skill.rainite.com` 的 DNS、Cloudflare Access、WAF、TLS 和挑战策略。
- 宿主机其他反向代理、代理软件或共享的 80/443 端口。

未经认证的命令行请求可能被 Cloudflare 挑战页以 HTTP 403 拒绝，这不表示本地应用部署失败。生产验收以 `127.0.0.1:19420` 的本地检查为基础，公开页面由已通过 Cloudflare 验证的浏览器复核。

不得把 Cloudflare Token、GitHub Token、SSH 私钥、Cookie 或其他凭据写入仓库、部署命令、日志或本文档。

## 持久数据

应用发布与用户数据严格分离。部署只替换正式二进制并重启 systemd，不运行 `skillshare sync`、`skillshare update` 或 `skillshare install`。

当前全局运行状态包括：

| 数据 | 路径或来源 |
| --- | --- |
| 全局配置 | `/root/.config/skillshare/config.yaml` |
| skills 源仓库 | 由全局配置决定；当前为 `/root/.config/skillshare/skills` |
| 安装元数据与 registry | skills 源目录及 Skillshare 配置目录内的对应文件 |
| 备份与回收站 | `/root/.local/share/skillshare/` |
| 状态与操作日志 | `/root/.local/state/skillshare/` |
| 可重建缓存 | `/root/.cache/skillshare/` |

构建、安装和回滚不得覆盖、清空、重新初始化或提交这些运行数据。修改 skills 内容是独立运维任务，应单独验证、提交和同步。

## 发布前提

正式构建前必须满足：

- 所有计划内代码和部署文档已提交并推送。
- 当前分支为 `main`，工作区和暂存区均干净。
- 本地 `HEAD` 与已获取的 `origin/main` 一致。
- Go、Node 和 pnpm 版本符合 `mise.toml` 与仓库 lockfile。
- 当前可用服务继续运行，候选二进制验证完成前不得覆盖正式二进制。

检查命令：

```bash
cd /root/skills/skillshare
git fetch origin
test "$(git branch --show-current)" = main
test -z "$(git status --porcelain)"
test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)"
```

## 验证顺序

所有发布验证串行执行。前一个步骤未成功退出时，不启动后一个步骤；命令长时间无输出时继续轮询原进程，不重复启动同一任务。

先验证 Go 代码：

```bash
cd /root/skills/skillshare
make fmt-check
GOCACHE=/tmp/skillshare-go-cache go vet ./...
GOCACHE=/tmp/skillshare-go-cache ./scripts/test.sh --unit
GOCACHE=/tmp/skillshare-go-cache ./scripts/test.sh --int
```

再验证 UI：

```bash
cd /root/skills/skillshare/ui
pnpm install --frozen-lockfile
pnpm exec tsc --noEmit
pnpm exec eslint .
pnpm exec vitest run --maxWorkers=1
```

普通 CLI/UI 发布不强制运行 Docker、red-team 或 website 验证。改动相应模块时必须追加其专项检查：供应链审计相关改动运行 `make test-redteam`，网站改动运行 `cd website && npm run build`。

`scripts/test.sh` 会重建 `bin/skillshare`，因此带版本信息的正式构建必须放在全部测试之后。

## 正式构建

生产二进制必须内嵌 UI，并使用当前干净提交的 `git describe` 结果作为版本。去掉标签开头的 `v`，但保留标签之后的提交数和短哈希；不得部署 `dev` 或带 `-dirty` 的版本。

```bash
cd /root/skills/skillshare

release_version="$(git describe --tags --always --dirty)"
release_version="${release_version#v}"
case "$release_version" in
  *-dirty) printf 'refusing dirty build: %s\n' "$release_version" >&2; exit 1 ;;
esac

make ui-stage-embed
GOCACHE=/tmp/skillshare-go-cache go build \
  -tags embedui \
  -ldflags "-X main.version=${release_version}" \
  -o bin/skillshare \
  ./cmd/skillshare
```

构建后执行 smoke test，并记录候选哈希：

```bash
test "$(./bin/skillshare --version)" = "skillshare v${release_version}"
./bin/skillshare --help >/dev/null
./bin/skillshare ui --help >/dev/null
sha256sum bin/skillshare
file bin/skillshare
```

## systemd 单元

唯一正式单元路径为 `/etc/systemd/system/skillshare-ui.service`，内容应为：

```ini
[Unit]
Description=Skillshare Web UI
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=root
Environment=HOME=/root
WorkingDirectory=/root
ExecStart=/usr/local/bin/skillshare ui --global --host 127.0.0.1 --port 19420 --no-open
Restart=on-failure
RestartSec=3s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

首次安装单元或修改单元内容后执行：

```bash
systemctl daemon-reload
systemctl enable skillshare-ui
systemctl restart skillshare-ui
```

普通二进制发布没有修改单元文件时，不需要 `daemon-reload`。

## 正式部署

以下流程会先保留当前正式二进制，再把候选文件安装到同一文件系统，使用 `mv` 原子替换。新服务未通过版本化健康检查时立即恢复旧二进制。

```bash
set -euo pipefail

release_repo=/root/skills/skillshare
release_source="${release_repo}/bin/skillshare"
release_target=/usr/local/bin/skillshare
release_stamp="$(date -u +%Y%m%dT%H%M%SZ)"
release_backup="${release_target}.backup-${release_stamp}"
release_candidate="${release_target}.candidate-${release_stamp}"
release_expected_version="$("$release_source" --version | sed -n 's/^skillshare v//p')"

test -n "$release_expected_version"
case "$release_expected_version" in
  dev|*-dirty) printf 'refusing non-release build: %s\n' "$release_expected_version" >&2; exit 1 ;;
esac
test -x "$release_source"
test -x "$release_target"
test ! -e "$release_backup"
test ! -e "$release_candidate"

cp -p "$release_target" "$release_backup"
install -m 0755 "$release_source" "$release_candidate"
mv "$release_candidate" "$release_target"

if ! systemctl restart skillshare-ui; then
  install -m 0755 "$release_backup" "$release_candidate"
  mv "$release_candidate" "$release_target"
  systemctl restart skillshare-ui
  exit 1
fi

release_healthy=false
release_health=
for release_attempt in 1 2 3 4 5 6 7 8 9 10; do
  if systemctl is-active --quiet skillshare-ui \
    && curl -fsS --max-time 3 http://127.0.0.1:19420/ >/dev/null; then
    release_health="$(curl -fsS --max-time 3 http://127.0.0.1:19420/api/health 2>/dev/null || true)"
    if printf '%s' "$release_health" | grep -Fq '"status":"ok"' \
      && printf '%s' "$release_health" | grep -Fq "\"version\":\"${release_expected_version}\""; then
      release_healthy=true
      break
    fi
  fi
  sleep 1
done

if [ "$release_healthy" != true ]; then
  journalctl -u skillshare-ui -n 50 --no-pager || true
  install -m 0755 "$release_backup" "$release_candidate"
  mv "$release_candidate" "$release_target"
  systemctl restart skillshare-ui
  exit 1
fi

printf 'deployed=%s\nbackup=%s\nhealth=%s\n' \
  "$release_expected_version" "$release_backup" "$release_health"
```

历史备份不自动删除。确认多个后续版本长期稳定后，只能显式选择具体备份文件进行人工清理；不要使用未核对的通配符删除。

## 健康检查与日志

每次安装、启动、重启、更新或回滚后执行：

```bash
systemctl is-enabled skillshare-ui
systemctl is-active skillshare-ui
systemctl status skillshare-ui --no-pager
systemctl cat skillshare-ui
ss -ltnp '( sport = :19420 )'
curl -fsS --max-time 5 -o /dev/null -w 'ui_http=%{http_code}\n' \
  http://127.0.0.1:19420/
curl -fsS --max-time 5 http://127.0.0.1:19420/api/health
sha256sum /root/skills/skillshare/bin/skillshare /usr/local/bin/skillshare
journalctl -u skillshare-ui -n 50 --no-pager
```

验收必须确认：

- 单元为 `enabled` 和 `active`。
- 只有一个正式 Skillshare UI 进程监听 `127.0.0.1:19420`。
- 首页返回 HTTP 200，内容类型为 HTML。
- `/api/health` 返回 `status: ok`，且 `version` 与候选二进制一致。
- 构建产物与正式二进制 SHA-256 一致。
- journal 显示旧进程优雅退出、新进程正常启动，没有持续重启或启动错误。
- 通过已完成 Cloudflare 验证的浏览器访问 `https://skill.rainite.com`，页面和 API 功能正常。

持续查看日志：

```bash
journalctl -u skillshare-ui -f
```

## 回滚

列出备份并人工选择最近一个确认可用的具体文件：

```bash
ls -lh /usr/local/bin/skillshare.backup-*
```

假设已确认回滚文件为 `/usr/local/bin/skillshare.backup-<UTC时间>`：

```bash
set -euo pipefail

rollback_source='/usr/local/bin/skillshare.backup-<UTC时间>'
rollback_candidate=/usr/local/bin/skillshare.rollback-candidate

test -x "$rollback_source"
test ! -e "$rollback_candidate"
install -m 0755 "$rollback_source" "$rollback_candidate"
mv "$rollback_candidate" /usr/local/bin/skillshare
systemctl restart skillshare-ui
```

回滚后执行完整健康检查并记录实际版本与二进制 SHA-256。问题修复、重新验证并发布前，不要覆盖已确认可用的备份。
