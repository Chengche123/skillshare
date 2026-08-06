# Repository Guidelines

## Project Structure & Module Organization
`cmd/skillshare/` 存放 CLI 入口和命令处理逻辑。核心 Go 包位于 `internal/`，包括 `config`、`install`、`sync`、`audit`、`server` 等。单元测试与源码同目录放置，命名为 `*_test.go`；端到端 CLI 测试位于 `tests/integration/`。前端相关代码分为 `ui/`（React + Vite 仪表盘）、`website/`（Docusaurus 文档站点）和 `video/`（Remotion 宣传视频）。辅助脚本和资源主要在 `scripts/`、`schemas/`、`.github/assets/` 和 `skills/skillshare/`。

## Build, Test, and Development Commands
推荐开发环境是运行 `make devc` 进入 devcontainer，其中已预装 Go、Node 和 pnpm。

- `make build`：构建 `bin/skillshare`。
- `make run`：运行本地二进制并显示 `--help`。
- `make test`：通过 `./scripts/test.sh` 运行单元测试和集成测试。
- `make check`：执行 format check、`go vet` 和测试，是提交 PR 前的最低检查标准。
- `make test-redteam`：运行供应链安全回归测试。
- `make ui-dev`：启动 Go API 和 Vite UI。
- `cd ui && pnpm run test`：运行 Vitest。
- `cd website && npm run build`：验证文档站点可正常构建。

## Coding Style & Naming Conventions
所有 Go 代码改动都应使用 `gofmt`，并遵循标准 Go 格式及小写包目录命名。命令相关逻辑应尽量靠近 `cmd/skillshare/` 中对应文件，优先拆分为小而明确的 `internal` 包，而不是堆积通用 helper。UI 代码延续现有 TypeScript/React 约定：组件和页面使用 `PascalCase.tsx`，hooks 使用 `use*.ts`，工具模块文件名保持小写。提交 UI 改动前运行 `cd ui && pnpm run lint`。

## Testing Guidelines
修改某个包时，应在对应目录补充或更新单元测试；涉及 CLI 行为时，应在 `tests/integration/` 中补充集成测试。可按需运行 `./scripts/test.sh --unit`、`./scripts/test.sh --int` 或 `./scripts/test.sh --cover`。CI 还会执行 `go test -race`、Docker sandbox 测试以及 red-team 检查。仓库没有明确的覆盖率门槛，但凡改动 `install`、`sync`、`audit` 相关流程，都不应在缺少测试覆盖的情况下提交。

## Commit & Pull Request Guidelines
提交信息遵循现有 Conventional Commit 风格，例如 `fix(ui): ...`、`feat(target): ...` 或 `docs: ...`。新增功能应先开 issue；较大的改动需基于 `proposals/TEMPLATE.md` 提交 proposal。PR 应保持聚焦，在适用时关联 issue，并确保 `make check` 通过。涉及 UI、文档或资源文件的可视化改动，建议附上截图或简短 GIF。

## Security & Configuration Tips
优先使用 devcontainer，或与 `mise.toml` 中固定版本保持一致（`go 1.25.5`、`pnpm 10.28.0`）。如果改动了 `install`、`audit` 或 sandbox 相关逻辑，务必运行 `make test-redteam`，并避免引入不必要的网络依赖测试。

## Deployment & Operations
根目录 `DEPLOYMENT.md` 是本项目构建、发布、生产部署和运行维护的唯一事实来源。执行任何正式构建、发布、部署、回滚或服务变更前必须完整阅读并遵循该文档。

- 构建命令、版本注入方式、二进制路径、systemd 单元、监听地址、端口、公开域名、健康检查、日志或回滚方式发生变化时，必须在同一任务和同一提交中更新 `DEPLOYMENT.md`。
- 正式发布必须按 `DEPLOYMENT.md` 的顺序完成 Go 与 UI 验证，再构建带内嵌 UI 和 `git describe` 版本信息的二进制；不得部署 `dev` 版本。
- 应用部署只替换程序二进制并重启 systemd 服务。不得把 `skillshare sync`、`update` 或 `install` 混入部署流程，也不得覆盖全局配置、skills 源仓库、registry、备份或其他用户数据。
- Cloudflare Tunnel、DNS 和访问策略由宿主机外部配置管理。不得在仓库、命令输出或部署文档中记录 Tunnel Token、Cloudflare 凭据或其他秘密。
- 每次经过验证的代码或部署文档变更都应创建聚焦的本地 Git 提交；GitHub 可用时推送到 `origin/main`。
