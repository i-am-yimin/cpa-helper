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

## 2026-11-09 生产部署验收

- 生产代码：`dee502800342fe1979119c4ae6ff8aa13187d297`；[构建与部署流水线通过](https://github.com/i-am-yimin/cpa-helper/actions/runs/42504587791)。
- 生产 Core 的 glibc 与新构建的插件匹配，插件已注册并有效启用，版本为 `1.0.0`。
- 一笔受控模型请求返回 HTTP 200；共享目录中的原生调度心跳已更新，证实插件参与了真实选路。
- 公网 `/api/ready` 返回 200，当前/目标数据库版本均为 `202611080001`；Looseleaf 字体资源返回 200。
- 未认证模型接口返回 401，公网管理接口继续隐藏（404）；已有管理员和设置保留。
- 上述生产验证补齐了此前的原生加载与真实请求验证；仍未执行 PostgreSQL 容器验收。

接入与恢复方式见 [Antigravity 分组调度说明](antigravity-group-scheduler.md)。设计规范见根目录 [DESIGN.md](../DESIGN.md)。本地截图位于 `.impeccable/review/`，不进入 Git。

## 2026-11-09 BoardUI 与 Codex 持久暂停改造

- 界面已采用 BoardUI 的 MIT 开源主题、排版令牌和组件样式，适配现有 Vue / Reka，覆盖登录、共享控件和管理页面。移除 Looseleaf CSS、Kevin Sans 和手绘描边；本地字体改为 Inter Variable，包含字体与 BoardUI 授权副本。
- Codex 低额度暂停、全局/独立/关闭阈值、实际 CPA 禁用、明确的手动恢复、人工禁用保护和手动优先级持久化均已实现。新规则默认关闭，未代用户填写启用百分比。详见 [暂停规则设计与实现](codex-quota-manual-pause-design.md)。
- 插件就绪状态改为读取 Core 注册表，近期调度心跳单独展示。新增测试验证插件空闲仍就绪、禁用不被旧心跳覆盖、读取失败为未知而非未安装。
- 本地后端全量 312 个测试事件通过，32 个外部条件用例跳过；随后新增的插件状态测试定向通过。发现并修复了刷新收尾越过数据库/日志生命周期的竞态，原失败用例连续 5 次通过。
- 全平台浏览器用例 51 项：50 项逐项通过，1 项需单独开启的实账导出用例跳过。SSO 过期回调的旧断言误拒绝合法的用量时间筛选 query，改为准确校验目标 pathname，3 项 SSO 用例定向复跑通过。
- `npm run lint`、`npm run test:i18n`、`npm run build`、6 项部署测试和 OIDC 绑定工具测试通过。SQLite 迁移、PostgreSQL SQL 链、导入表清单、启动就绪版本同步为 `202611090001`；未执行 PostgreSQL 实库验收。
- 1440/768/390 视图、深色模式、全局规则及暂停明细截图已检查。列表按 959px 内容宽度切换卡片行，避免侧栏挤压后隐藏启用开关。独立视觉复核认可 BoardUI 替换与布局；两项对比度修复经复核均为 resolved，最终 disposition 为 `ship`（该轮结论范围为所列修复）。
- 设计令牌 YAML 与 schema-v2 sidecar 通过校验，9 个片段已在 Chrome shadow DOM 中渲染验证。截图和测试数据均来自隔离环境，未设置生产阈值或修改生产凭证状态。

### 本次生产验收

- 生产代码 `544b67ebf2e532da1bbf8927eed578152c014acb`；[Deploy AI Gateway #42519174069](https://github.com/i-am-yimin/cpa-helper/actions/runs/42519174069) 的构建和部署均成功。镜像 digest 为 `sha256:a2597b574e4cd8ad7f71d761b5d51496f187b0ded236aee6642483d94659f1d8`。
- 公网 `/api/ready` 返回 200，当前与目标版本均为 `202611090001`。生产 CSS 包含 BoardUI 主题和 Inter；实际 WOFF2 字体返回 200、48,256 bytes，BoardUI 与字体授权资源均可读取。
- Core 确认原生分组插件已注册、有效启用，版本 `1.0.0`；一笔受控请求返回 HTTP 200，并更新原生调度心跳。已有管理员和设置保留。
- 未认证的额度规则接口与模型接口均返回 401；公网 V8 管理接口返回 404。生产登录按现有配置跳转 OIDC 身份服务。
- 前一构建在发布前被“过期提交”检查拦截，没有覆盖较新的修复；最终部署提交包含凭证异常及健康恢复过程中继续维持暂停 `-1` 的回归保护。
