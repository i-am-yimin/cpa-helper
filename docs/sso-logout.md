# CAS / OIDC 退出与重新登录

生产站点使用 `cas.nigulas.top` 的原生 OIDC。正常登录和退出后的重新登录都应在统一身份服务完成。

## 已修复的问题

旧版 OpenResty 对 `/login` 无条件重定向，忽略了退出后的 `skipsso=true`。修复代理后，退出落点已改到 CAS，但真实已登录会话仍可能立即授权回来。

2026-11-09 用户提供的 HAR 确认：网关已撤销会话，新的授权也带有 `prompt=login`，CAS 却直接返回新授权码。原因是 CPA OIDC Provider 继承了 Nginx UI 的 `default-provider-invalidation-flow`；该流程没有用户退出阶段，只撤销应用令牌，保留 CAS 登录会话。authentik 根据已有登录事件判断 `prompt=login` 是否已完成交互，因此不能用该参数代替实际注销。

本版保留显式恢复入口，退出流程改为：

1. 清除网关登录 Cookie，并删除该 OIDC 会话的服务器记录。
2. 使用该会话对应的身份服务 end-session endpoint；CPA Provider 使用 `default-invalidation-flow`，其中的 `default-invalidation-logout` 用户退出阶段销毁当前浏览器的 CAS 登录会话。保留退出 state 校验。
3. 退出回调转到 `/oidc/login?prompt=login`，创建新的 state、nonce 和 S256 PKCE 授权，要求 CAS 交互登录。
4. 用户停留在 CAS，完成登录后才回到平台。身份服务没有退出端点或会话记录已过期时，也走此交互登录入口。

退出时设置安全的 HttpOnly 重新登录标记。刷新 CAS 页面、返回普通网关入口或重放旧回调不能退回静默登录；只有成功的交互式 OIDC 回调会清除标记。普通首次访问仍可使用原有 SSO 行为。

`/login?skipsso=true` 仅为显式本地恢复入口，不再作为正常退出落点。公网本地密码 API 仍被 OpenResty 拦截。

注册脚本显式选择会话退出流程，不再继承参考应用的退出语义。每次部署在保留现有配置之前执行 `configure-sso.py reconcile-logout`，只修改已核对身份的 CPA Provider 的 `invalidation_flow`；不修改共享流程或其他应用。部署健康检查验证 Provider 绑定及流程内真实的用户退出阶段，发现配置漂移时失败。

这会结束当前浏览器的统一身份会话；其他设备的 CAS 会话不受该阶段影响。

## 验证

- 后端全量：316 个测试事件通过，32 个依赖外部环境的测试跳过。
- SSO 浏览器回归：3 项通过，使用真实侧栏退出操作；模拟 CAS 的已有会话可以忽略 `prompt=login`，要求 end-session 删除中央会话并清除 Cookie，才会显示交互表单。
- 部署回归新增 5 项，覆盖应用退出流程修复、幂等执行、缺失用户退出阶段、Provider 身份不匹配、健康检查与未注册安装。
- 覆盖退出后的 CAS 停留、刷新、重新打开普通入口、显式重新登录、会话撤销、nonce/PKCE、重放保护与重新登录标记清除。
- 生产 CI 增加独立 SSO 浏览器阶段，使用隔离数据库和本地模拟身份服务，不依赖真实账号密码。

身份服务依据：生产使用的 [authentik authorize](https://github.com/i-am-yimin/authentik-infra/blob/main/src/authentik/authentik/providers/oauth2/views/authorize.py)、[应用退出流程](https://github.com/i-am-yimin/authentik-infra/blob/main/src/authentik/blueprints/default/flow-default-provider-invalidation.yaml)、[会话退出流程](https://github.com/i-am-yimin/authentik-infra/blob/main/src/authentik/blueprints/default/flow-default-invalidation-flow.yaml)及 [UserLogoutStage](https://github.com/i-am-yimin/authentik-infra/blob/main/src/authentik/authentik/stages/user_logout/stage.py)。

## 前次生产检查的局限（2026-11-09）

代码 `2c691194c198d8491ca4a74c34b1de06695e9803` 的[流水线](https://github.com/i-am-yimin/cpa-helper/actions/runs/42540875272)通过，浏览器能到达 CAS 登录表单，但当时没有验证已有 CAS 登录会话的真实退出。用户随后提供的 HAR 证明该检查不足以确认问题修复。新的生产验收必须先确认网关和 CAS 均已登录，再执行退出，确认中央会话失效、CAS 返回交互表单、网关保持未登录，重新打开普通入口也不能自动回到账号。

## 会话退出修复的生产验收（2026-11-09）

代码 `ce9005470e618bb6126e4cbf2059d372a3fccb09` 的[构建与部署流水线](https://github.com/i-am-yimin/cpa-helper/actions/runs/42542559922)通过，包含前后端及部署回归；线上健康检查确认 CPA Provider 使用包含用户退出阶段的会话退出流程。

使用用户 HAR 中仍有效的会话验收，未记录 Cookie、令牌或提交账号密码：

- 退出前网关 `/api/auth/me` 为 200，CAS 用户会话已认证。
- 调用真实退出接口后，CAS 执行 `default-invalidation-flow`，经校验回调进入默认认证流程，返回 `ak-stage-identification` 交互表单。
- 退出后网关为 401，CAS 为未登录；重放旧 CAS Cookie 返回 403。
- 重新打开普通 `/oidc/login` 仍停在 CAS 认证流程，网关继续为 401。

另外在生产浏览器已登录账号上，通过实际侧栏菜单点击“退出登录”，确认到达 CAS 的“登录以继续访问 AI Gateway”账号密码表单；刷新仍保留表单，没有自动回到平台。浏览器可能填充保存的账号密码，验收没有点击登录。
