# Antigravity 分组额度与优先级

统一管理入口为 `/admin/upstreams`。旧的 `/admin/credential-mgmt` 跳转到该页的 OAuth 标签，原有授权、文件导入、编辑、启停、删除和 Codex 巡检仍可使用。

## 为什么需要调度插件

CPA 的 auth-file `priority` 是凭证级字段，不能区分同一 Antigravity 凭证内的 Gemini、Claude / GPT、Argon 等额度组。本实现不复制 OAuth 凭证，不修改所有模型共用的 priority；由原生插件在 `scheduler.pick` 中，按照请求模型选择该凭证对应的分组优先级。

接口与 ABI 核对基准：CPA Core LTS **v1-lts-0.1.2**，ABI 1 / schema 7，启用 `scheduler_across_priorities`，以便优先级较低的 Antigravity 凭证仍可按分组配置与 Codex 比较。插件仅从 Core 提供的可用候选中选择，不扩大模型支持范围，不代替 Core 的鉴权、冷却或模型注册逻辑。其他版本须先确认此能力。

## 分组规则

- 每个“凭证名称 + 上游 group ID”独立保存优先级和模型匹配规则。上游未给 ID 时使用分组名称；上游改名后需重新核对映射。
- 初始优先级继承凭证 priority。可独立设定；数值越大越优先。`-1` 保留给自动额度降级，手动固定低优先级用 `< -1`。
- 任一有效窗口的已用比例达到“巡检与默认规则”的额度阈值，该组当前优先级降为 `-1`。所有窗口恢复后回到配置值；手动高优先级也受额度约束。
- 启用只检查不修改（Dry Run）时不自动降低分组优先级。保存优先级本身是显式配置操作。
- 额度失败、未知重置时间、过期观察不用于自动降级。观察有效期最多 5 分钟，且不跨重置时间。观察过期后回退到配置的分组优先级，Core 自身的模型冷却仍然有效。
- 模型名精确匹配优先于通配符；更具体的通配符优先。相同具体程度同时命中多个组时不猜测，回退凭证优先级，也不把请求归入任意一个组。
- 内置 `Gemini Models → gemini-*`、`Claude and GPT models → claude-*, gpt-*`、Argon 组 `gemini-4*, argon*` 是可编辑的初始映射；必须按实际模型 ID 核对。别名应填写 Core 在 scheduler 中使用的实际模型名。未知组需要显式配置。
- 插件同优先级默认轮询；Core 设置为 fill-first 时，将下述插件环境变量也设为 `fill-first`。不实现 weighted-round-robin，请勿在需要该策略的部署中将此插件当作等价替换。

## 窗口统计口径

复用 Codex 的本地 usage records、Token 汇总和实际计价逻辑，按凭证身份、模型分组、`[reset - window, reset)` 区间计算请求、Token、费用、未计价请求。短窗口与周窗口会重叠，不能相加。更具体分组优先，避免 Gemini 与 Argon 重复归因。缺少时长、重置时间、模型映射或读取失败时显示未知，不伪造 0。

百分比来自上游额度，费用仅来自本平台已采集的请求；没有把本地美元消耗线性外推为 Antigravity 总额度。

## 构建和接入

在与 Core **相同 CPU 架构和 libc** 的 Linux 构建环境中，从 `backend` 执行：

```sh
CGO_ENABLED=1 go build -buildmode=c-shared -o cpa-group-scheduler.so ./cmd/cpa-group-scheduler
```

或使用仓库根目录作为 Docker 构建上下文：

```sh
docker build -f deploy/group-scheduler/Dockerfile --output ./release/group-scheduler .
```

本次本地生成的 `release/cpa-group-scheduler.so` 为 Linux amd64 **musl** 版本，适用于相应 Alpine 环境。源码与测试在仓库内；构建产物不进入 Git。

1. 将 `.so` 以 **cpa-group-scheduler.so** 文件名放入 Core 配置的 `plugins.dir`。
2. 给 Helper 和 Core 挂载同一个可读写目录，例如 `/shared/cpa-helper`，并配置兼容的 UID / 文件权限。该目录承载无访问令牌的规则快照和调度心跳，不挂载 OAuth 文件。
3. Helper 环境变量：`CPA_HELPER_SCHEDULER_SNAPSHOT=/shared/cpa-helper/group-scheduler.json`。
4. Core 环境变量：`CPA_GROUP_SCHEDULER_SNAPSHOT=/shared/cpa-helper/group-scheduler.json`。可选 `CPA_GROUP_SCHEDULER_STRATEGY=round-robin` 或 `fill-first`。
5. 将以下配置合入 Core，保留已有插件和权限配置，使用 V8 强 ETag 编辑流程，勿替换整个生产配置文件：

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    cpa-group-scheduler:
      enabled: true
      priority: 1
```

Core 只启用一个最高顺序的 scheduler 插件。若已有调度插件，应先核对是否冲突；此插件无需 auth-list、auth-read、auth-write 或 model-execute 权限。

6. 启动后打开“上游与凭证”，展开 Antigravity 保存分组模型映射与优先级。首次保存或查看额度会生成快照，后台每分钟检查并在上游缓存到期时刷新，关闭浏览器后继续工作。
7. 发起一笔命中该分组的受控请求。插件在真正参与选择后写入 `.status` 心跳；管理页只有在近 2 分钟存在该证据时显示“分组插件已参与调度”。无新请求超过 2 分钟会显示等待验证，不能据此直接判断插件进程已停。

禁用该 scheduler 插件即可回到 CPA 内置调度；未改写全局 priority，也未复制或拆分 OAuth 凭证。共享文件缺失、损坏、无法匹配模型时回退内置选路。

2026-11-09 已部署至 [AI Gateway](https://aigateway.nigulas.top)。生产镜像使用与 Core 匹配的 Linux glibc 插件，Core 已确认插件注册并有效启用；一笔受控模型请求返回 HTTP 200，并产生新的原生调度心跳。部署保留了现有管理员、设置和账户数据。记录见 [Deploy AI Gateway #42504587791](https://github.com/i-am-yimin/cpa-helper/actions/runs/42504587791)，生产代码提交为 `dee502800342fe1979119c4ae6ff8aa13187d297`。

## 上游依据

- [CPA Core scheduler 示例](https://github.com/BlueSkyXN/CPA-Core-LTS/tree/v1-lts-0.1.2/examples/plugin/scheduler)
- [候选选择与属性合同](https://github.com/BlueSkyXN/CPA-Core-LTS/blob/v1-lts-0.1.2/sdk/cliproxy/auth/conductor_selection.go)
- [RPC 注册能力](https://github.com/BlueSkyXN/CPA-Core-LTS/blob/v1-lts-0.1.2/internal/pluginhost/rpc_schema.go)
- [Looseleaf 源码与字体授权](https://github.com/kevintngo/looseleaf)
