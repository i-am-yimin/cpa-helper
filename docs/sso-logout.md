# CAS / OIDC 退出与重新登录

生产站点使用 `cas.nigulas.top` 的原生 OIDC。正常登录和退出后的重新登录都应在统一身份服务完成。

## 已修复的问题

旧版 OpenResty 对 `/login` 无条件重定向，忽略了退出后的 `skipsso=true`，会立即再次进入授权。上一轮代理修复阻止了这个循环，却让退出落点停在网关本地登录表单，不符合统一登录要求。

本版保留显式恢复入口，退出流程改为：

1. 清除网关登录 Cookie，并删除该 OIDC 会话的服务器记录。
2. 使用该会话对应的身份服务 end-session endpoint；保留退出 state 校验。
3. 退出回调转到 `/oidc/login?prompt=login`，创建新的 state、nonce 和 S256 PKCE 授权，要求 CAS 交互登录。
4. 用户停留在 CAS，完成登录后才回到平台。身份服务没有退出端点或会话记录已过期时，也走此交互登录入口。

退出时设置安全的 HttpOnly 重新登录标记。刷新 CAS 页面、返回普通网关入口或重放旧回调不能退回静默登录；只有成功的交互式 OIDC 回调会清除标记。普通首次访问仍可使用原有 SSO 行为。

`/login?skipsso=true` 仅为显式本地恢复入口，不再作为正常退出落点。公网本地密码 API 仍被 OpenResty 拦截。

## 验证

- 后端全量：316 个测试事件通过，32 个依赖外部环境的测试跳过。
- SSO 浏览器回归：3 项通过，使用真实侧栏退出操作；模拟身份服务保留会话，只有 `prompt=login` 才显示交互表单。
- 覆盖退出后的 CAS 停留、刷新、重新打开普通入口、显式重新登录、会话撤销、nonce/PKCE、重放保护与重新登录标记清除。
- 生产 CI 增加独立 SSO 浏览器阶段，使用隔离数据库和本地模拟身份服务，不依赖真实账号密码。

身份服务对 `prompt=login` 的处理依据：[authentik OAuth2 authorize 实现](https://github.com/goauthentik/authentik/blob/main/authentik/providers/oauth2/views/authorize.py)。
