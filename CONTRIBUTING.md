# Contributing to skillshare

感谢你关注并参与贡献。这份指南用于帮助你快速进入正确的协作方式。

## Start with an Issue

最合适的贡献方式，是先[提交 issue](https://github.com/runkids/skillshare/issues/new)。idea、功能请求和设计讨论都应先在 issue 中进行，这样可以帮助我们：

- 先确认改动是否符合项目方向
- 在写代码前先对范围和方案达成一致
- 避免把时间投入到最终不会合并的改动上

**请先开 issue，再开始写代码**，即使你已经非常确定实现方案。跳过这一步，是贡献无法被接受的最常见原因。

## Pull Requests

### What PRs are good for

PR 最适合处理**小而聚焦的改动**：

- 可以稳定复现的 bug 修复
- 拼写错误和文档修正
- 小型改进（涉及文件较少，且有效改动通常少于约 200 行）

这类改动可以直接提 PR；如果已有对应 issue，仍然应当关联上。

### Feature Ideas

对于新功能或较大的改动：

1. 先**提交 issue**，说明你的想法以及它要解决的问题
2. 再**提交 proposal**：复制 [`proposals/TEMPLATE.md`](proposals/TEMPLATE.md)，填写后向 `proposals/` 提交 PR

通过的 proposal 会被加入 roadmap。实际实现通常由 maintainer 负责，以保证与项目架构和代码约定保持一致。

> **Note:** 由于这个项目的性质，大多数功能型 PR 不会被直接合并。但每一份贡献仍然有价值，你的 PR 会作为具体参考，影响最终实现方向。

### PR Checklist

- [ ] 已关联 issue（功能改动必需，bug 修复建议关联）
- [ ] 已补充测试且测试通过（`make check`）
- [ ] diff 中不包含无关改动
- [ ] commit message 解释了“为什么改”，而不只是“改了什么”
- [ ] PR 范围保持聚焦，一个 PR 只处理一个问题

## Development Setup

所有开发和测试工作都应在 **devcontainer** 中完成。这样可以确保环境一致，避免本地工具链差异带来的问题；其中 Go toolchain、Node.js、pnpm 和演示内容都已预先配置。

```bash
git clone https://github.com/runkids/skillshare.git
cd skillshare
make devc            # start devcontainer + enter shell (one step)
```

进入 devcontainer 后，可使用：

```bash
make build           # 构建二进制 → bin/skillshare
make test            # 运行 unit + integration tests
make check           # 运行 fmt + lint + test（PR 前必须通过）
make ui-dev          # 启动 Go API server + Vite HMR for Web UI
```

其他常用 devcontainer 命令：

```bash
make devc-down       # 停止 devcontainer
make devc-restart    # 重启 devcontainer
make devc-reset      # 完整重置（删除 volumes）
make devc-status     # 查看 devcontainer 状态
```

## Questions?

如果你不确定该从哪里开始，可以先查看[现有 issues](https://github.com/runkids/skillshare/issues)，或新开一个 issue 讨论你的想法。
