## graphify

本项目在 `graphify-out/` 下维护了 graphify 知识图谱。

规则：
- 在回答架构或代码库问题前，先阅读 `graphify-out/GRAPH_REPORT.md`，了解高连接节点（god nodes）和社区结构。
- 如果存在 `graphify-out/wiki/index.md`，优先从该 wiki 导航，而不是直接阅读原始文件。
- 在本次会话中修改代码文件后，运行 `graphify update .` 以保持图谱最新（仅做 AST 提取，无 API 成本）。
- 如果 `graphify update .` 的报错中明确包含 `too large for HTML viz`，可以忽略该报错；这表示 HTML 可视化产物过大，不影响前面的 AST 提取结果。其他 graphify 错误不要忽略。