# Skill Custom Groups

Date: 2026-04-24

## Summary

为 skill 增加用户自定义分组功能。分组信息存放在 skillshare 的系统元数据中，不写入 `SKILL.md`。`/resources?tab=skills` 页面可以按分组过滤 skills，同一个 skill 可以属于多个分组。

该功能只用于资源页归类和过滤，不影响同步、禁用、`.skillignore`、target include/exclude、audit、install 或 update 行为。

## Problem

当前资源页可以按来源类型过滤 skills，例如 `All`、`Tracked`、`GitHub`、`Local`，也可以按目录视图查看 `relPath` 形成的结构。但用户无法把“不常用”“备用”“实验中”“团队内”等跨目录、跨仓库的 skill 放到自定义集合里。

把分组写入 `SKILL.md` 不合适，因为这些分组是用户本地组织信息，不属于 skill 内容本身，也不应该随着 skill 发布或同步给其他用户。项目已有 `skills/.metadata.json` 作为系统元数据文件，因此分组应在该层保存。

## Goals

- 在资源页 skills tab 支持按自定义分组过滤 skills。
- 支持一个 skill 属于多个自定义分组。
- 支持 tracked repo 内的每个 skill 独立加入分组。
- 分组信息写入系统元数据，不修改 `SKILL.md`。
- 对没有安装来源 metadata 的本地 skill，也可以保存分组。
- 分组只影响 UI 归类和过滤，不改变现有同步和禁用语义。

## Non-Goals

- 不为 agents 增加自定义分组。
- 不新增 CLI 分组管理命令。
- 不新增独立的分组管理页面。
- 不让分组影响 sync、disable、target filtering、audit 或 install/update。
- 不把自定义分组写入 skill frontmatter。
- 不把自定义分组和现有目录分组、tracked repo 分组、`MetadataEntry.Group` 合并。

## Confirmed Decisions

- 分组只用于 `/resources?tab=skills` 页面归类和过滤。
- 分组管理入口只放在资源页。
- 输入新分组名时自动创建分组；当没有 skill 使用某个分组时，该分组自然消失。
- 过滤语义为单选分组：未选择分组时显示全部，选择一个分组时显示属于该分组的 skills。
- tracked repo 内的每个 skill 都可以独立加入自定义分组。
- 分组信息存入现有 `skills/.metadata.json`。

## Data Model

扩展 `internal/install.MetadataEntry`，新增字段：

```go
CustomGroups []string `json:"custom_groups,omitempty"`
```

现有 `MetadataEntry.Group` 保持原意：表示安装路径或嵌套路径的 group。它不能用于自定义分组，以免和目录/仓库结构混淆。

metadata key 使用 full relative path，也就是 discovery 返回的 `DiscoveredSkill.RelPath`。这样可以正确区分：

- top-level skill: `my-skill`
- nested skill: `team/tools/my-skill`
- tracked repo child skill: `_repo-name/path/to/my-skill`
- 不同路径下同名 skill

读取时继续允许 `GetByPath(relPath)` 兼容旧 basename key。写入分组时迁移或创建 full-path key。

## Metadata Entry Lifecycle

自定义分组需要支持没有安装来源 metadata 的本地 skill。因此保存分组时可以创建轻量 entry，例如：

```json
{
  "version": 1,
  "entries": {
    "local-skill": {
      "custom_groups": ["unused", "reference"]
    }
  }
}
```

清空分组时：

- 如果 entry 仍有安装来源字段，例如 `source`、`repo_url`、`type`、`tracked`、`branch`、`file_hashes` 等，则只清空 `custom_groups` 并保留 entry。
- 如果 entry 只承载自定义分组，清空后删除该 entry，保持 `.metadata.json` 干净。

reconcile/prune 逻辑不能因为 entry 没有 `source` 就删除它。只有对应 skill 目录不存在时，才应该清理该 entry。

## API Design

### List Resources

`GET /api/resources` 返回的 skill item 增加字段：

```json
{
  "groups": ["unused", "reference"]
}
```

只对 `kind: "skill"` 返回该字段。没有分组时通过 `omitempty` 省略字段，前端统一归一化为空数组处理。

`handleListSkills` 从 `skillsStore.GetByPath(d.RelPath)` 读取 `CustomGroups`，并挂到 `skillItem.Groups`。

### Update Skill Groups

新增接口：

```http
PATCH /api/resources/{name}/groups
Content-Type: application/json

{
  "groups": ["unused", "reference"]
}
```

行为：

- `{name}` 优先按 `DiscoveredSkill.FlatName` 匹配，再兼容 basename 匹配。
- 写入时使用匹配到的 `RelPath` 作为 metadata key。
- `groups: []` 表示清空所有自定义分组。
- 找不到 skill 返回 404。
- 对 agent 或 `kind=agent` 调用返回 400，错误信息说明 custom groups only support skills。
- 保存成功后写一条 oplog，例如 `set-skill-groups`，字段包含 `name`、`groups`、`scope: "ui"`。

前端 API helper 增加：

```ts
setSkillGroups(name: string, groups: string[]): Promise<{ success: boolean }>
```

## Validation

服务端负责归一化和校验，前端可以做即时提示但不能替代服务端。

规则：

- 对每个分组名执行 trim。
- 空字符串丢弃。
- 去重。
- 保存时按字母排序，减少 metadata diff 噪声。
- 单个分组名最长 64 个字符。
- 每个 skill 最多 20 个分组。
- 允许中文、英文、数字、空格、`-`、`_`、`.`。
- 禁止 `/`、`\` 和控制字符，避免和路径语义混淆。

校验失败返回 400，并带明确错误信息。metadata 解析失败时不能覆盖原文件，应返回 500 或沿用现有 metadata load error 处理。

## Frontend Design

资源页只在 `activeTab === "skills"` 时显示分组功能。agents tab 不显示分组过滤，也不能编辑分组。

### Group Filter

在现有 sticky toolbar 中加入一个分组下拉：

- 默认值为 `All groups`。
- 选项从当前 skills 的 `groups` 聚合得出。
- 选项按名称排序。
- 每个选项显示该组下 skill 数量。
- 切换到 agents tab 时隐藏分组过滤，并将当前 group filter 重置为 `All groups`。

过滤顺序可以保持前端现有模型：文本搜索、来源类型过滤、分组过滤、排序。分组过滤和 `All / Tracked / GitHub / Local` 是 AND 关系。

### Edit Groups

编辑入口放在 skill 的右键菜单中，新增 `Edit groups`。

交互：

- 点击后打开 `SkillGroupsEditor` 弹窗。
- 当前分组以 chip 形式展示，可删除。
- 输入组名后按 Enter 添加。
- 输入不存在的组名会自动创建。
- 保存时 PATCH 完整 groups 数组。
- 清空所有 chip 并保存表示该 skill 不属于任何自定义分组。

### Display

skill 卡片和表格行可以展示前 1 到 2 个分组 badge，超出显示 `+N`。展示应避免明显改变卡片高度；必要时使用固定区域或单行截断。

## Frontend Data Flow

`Skill` 类型增加：

```ts
groups?: string[];
```

资源页数据流：

- 原始列表来自 `api.listSkills(undefined, { includeContent: true })`。
- `groupOptions` 从 `skills.filter((s) => s.kind !== "agent")` 聚合。
- `filtered` 在现有搜索和来源过滤之外增加 `selectedGroup === "" || s.groups?.includes(selectedGroup)`。
- 编辑成功时乐观更新 React Query cache 中对应 skill 的 `groups`。
- 保存失败时回滚乐观更新，并展示错误 toast。
- mutation settle 后 invalidate `queryKeys.skills.all`，保持和现有资源操作一致。

## Error Handling

- 找不到 skill：404。
- agent 调用 groups API：400。
- 非法分组名或数量超过限制：400。
- metadata load/save 失败：500。
- UI mutation 失败：回滚乐观更新，并显示服务端错误。
- metadata 文件损坏时，不能以空 store 覆盖原文件。

## Code Touch Points

后端：

- `internal/install/metadata.go`
- `internal/server/handler_skills.go`
- 新增或扩展 `internal/server` groups handler
- `internal/server/server.go`
- 相关 metadata/reconcile helper

前端：

- `ui/src/api/client.ts`
- `ui/src/pages/ResourcesPage.tsx`
- 新增 `SkillGroupsEditor` component
- i18n 文案

文档：

- 资源管理或 skill 管理相关 docs，说明自定义分组只影响 UI 过滤。

## Tests

后端测试：

- `MetadataEntry.CustomGroups` JSON 读写兼容旧 metadata。
- `GET /api/resources` 返回 skill groups。
- 无 metadata 的本地 skill 可以通过 PATCH 创建轻量 entry。
- tracked repo 内 skill 可以独立保存 groups。
- 同名不同路径 skill 使用 `relPath` 区分。
- 清空 groups 时按 entry 类型保留或删除 metadata entry。
- 非法组名、超过数量、agent 调用、找不到 skill 均返回正确错误。

前端测试：

- `Skill` 类型和 API helper 支持 `groups`。
- 资源页聚合 group options。
- skills tab 分组过滤正常工作。
- 分组过滤与搜索、来源类型过滤是 AND 关系。
- 编辑分组成功后乐观更新，失败后回滚。
- agents tab 不显示分组过滤。

## Rollout Notes

该改动是向后兼容的。旧 `.metadata.json` 没有 `custom_groups` 字段时正常加载。新增字段使用 `omitempty`，没有分组的 skill 不产生额外 JSON 噪声。

实现完成后需要按项目规则运行常规测试。由于会修改代码文件，还需要运行：

```sh
graphify update .
```

如果该命令报错中明确包含 `too large for HTML viz`，可以忽略该可视化产物过大的错误；其他错误不能忽略。
