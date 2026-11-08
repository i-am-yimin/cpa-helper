# 上游与凭证重构验证

验证日期：2026-11-08。界面截图与浏览器测试均使用模拟数据或隔离的本地测试数据库，未操作生产服务。

## 结果

- `go test ./... -timeout=10m` 全量通过：294 个测试事件通过，32 个需要外部条件的测试跳过。随后新增的坏凭证隔离测试、分组策略权限断言和相关迁移就绪检查已定向通过。
- `npm run lint`、`npm run test:i18n`、`npm run build` 通过。
- Antigravity 额度、V8 配置保护、统一上游与旧有平台交互四组浏览器测试共 20 项逐项通过。最后的全平台长用例在更新纸张样式的高度、边框与页签断言后定向复跑通过。
- 覆盖分组保存、GPT 耗尽与恢复、Gemini 独立优先级、模型映射具体程度/歧义、窗口边界与凭证归因、未知数据、原生 API 启停及模型排除规则、V8 revision、草稿防刷新、旧入口跳转、只读权限、1440/768/390 宽度及稳定暗色主题。
- 独立复核结论为 ship；发现的墨线定位、坏凭证刷新隔离、原生密钥启停和草稿稳定性问题已修复。
- `DESIGN.md` 的 YAML 与 `.impeccable/design.json` 的 schema v2、8 个组件片段已校验。

## 插件产物

`release/cpa-group-scheduler.so`：Linux amd64 musl，4,635,936 bytes。

SHA-256：`b6a4743643a68748fd84710549704f5a57f23f615f69350699d6fd30a4eddfad`

已成功交叉编译并检查 `cliproxy_plugin_init`、`cliproxyPluginCall`、`cliproxyPluginFree`、`cliproxyPluginShutdown` 导出。纯 Go 协议和模拟候选选路测试通过；未在真实 CPA Core 进程加载，也未验证生产权限、真实账户额度或 PostgreSQL 容器。

接入与恢复方式见 [Antigravity 分组调度说明](antigravity-group-scheduler.md)。设计规范见根目录 [DESIGN.md](../DESIGN.md)。本地截图位于 `.impeccable/review/`，不进入 Git。
