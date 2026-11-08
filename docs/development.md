# 本 fork 的开发说明

## 分支与上游

本 fork 来自 `HanadaLee/cpa-helper`，后者来自 `walkingddd/CPA-Helper`。
以下状态更新于 2026-11-08：

| 分支 | 用途与状态 |
| --- | --- |
| `main` | 同步 `walkingddd/CPA-Helper:main`，当前提交 `4723a47`（2026-10-10）。本地 `main` 跟踪 `origin/main`。 |
| `ext` | HanadaLee 持续维护的扩展版本，最后提交 `110ee33`（2026-11-03）；作为本次适配基线。精简前，它比当时的 `main`（`0560f9e`）多 223 个提交。 |
| `chore/repository-slimdown` | 基于 `ext` 的仓库、依赖和构建精简。 |

早期 CPAMC 功能分支 `cpamc` 已按要求从本 fork 删除。
`main`、`ext` 和精简分支保留，便于比较和维护。Git 会在分支间共享对象，
这次完整克隆的 Git 对象包约 5.54 MiB，删分支并不会按源码目录大小减少空间。

本地远程配置：`origin` 指向 `https://github.com/i-am-yimin/cpa-helper.git`，
`upstream` 指向 `https://github.com/HanadaLee/cpa-helper.git`，
`walkingddd` 指向 `https://github.com/walkingddd/CPA-Helper.git`。
两个上游远程的 push URL 均设为 `DISABLED`，日常推送目标为自己的 `origin`。
后续在自己的功能分支同步上游：

```powershell
git fetch upstream
git merge upstream/ext
```

`main` 用来同步 walkingddd 的基础版本；本 fork 的适配改动在自己的功能分支维护。
同步 `main` 可使用 `git fetch walkingddd main` 后，在本地 `main` 上执行
`git merge --ff-only walkingddd/main`，再 `git push origin main`。
以上推送只修改自己的 fork，GitHub 的 fork 关系不会自动向上游同步代码。

## 本地开发

项目目录为 `D:\code\ai\cpa`，直接包含前后端源码。
前端使用 Node.js 22.13+（`frontend/.nvmrc` 指定 22 系列），Go 最低版本见 `backend/go.mod`。

```powershell
# 终端 1，从项目根目录进入后端
cd backend
go run ./cmd/cpa-helper

# 终端 2，从项目根目录进入前端
cd frontend
npm ci
npm run dev
```

默认后端 `http://127.0.0.1:18317`，前端 `http://127.0.0.1:5173`。
首次访问引导创建管理员；实际接入 CPA 需要在系统设置中配置服务地址和管理密钥。
程序运行数据位于 `data/`，依赖、构建产物、测试报告和本地环境变量不纳入 Git。

Go 1.21+ 支持自动下载所需工具链。若本地版本较旧，在当前终端使用
`$env:GOTOOLCHAIN='auto'` 后执行项目命令即可；无需修改系统 Go 安装。
本次验证使用临时目录中的 Node.js 22.23.4 和已缓存的 Go 1.26.5，未修改系统安装。
项目仍需要开发者准备符合要求的 Node.js。

## 精简内容

- 移除日常安装不需要的 `shadcn-vue` 组件生成 CLI，以及它引入的依赖；
  `src/components/ui` 和 `components.json` 保留，现有界面功能不变。
  如需生成新组件，可在 `frontend` 目录按需运行
  `npx shadcn-vue@2.8.2 add <组件名>`。
- 去掉强制将整个 Unovis 入口划入独立包的配置，让 Rollup 随懒加载路由分包。
  原配置将共享 Vue 代码也拉入图表包，使应用入口提前加载图表代码；现在用量页面按需加载图表。
- Docker 构建上下文仅允许 `Dockerfile`、`VERSION`、前后端源码，
  再排除依赖、产物、测试报告、测试源码和环境变量文件。
  文档、截图与迁移脚本仍保留在仓库，供开发和迁移使用。
- 移除原作者专用的 GitLab/Harbor 流水线。GitHub 发布改为手动触发，
  推送当前仓库所有者的 GHCR 镜像，不依赖原作者的 Docker Hub 凭据。
  Linux、Windows、macOS、FreeBSD 的原有二进制发布目标保留。
- 新增前端检查流水线；Docker 和 CI 使用 Node.js 22。
  Compose 默认构建本 fork 的源码，避免仍运行原作者的镜像。

## 体积对比

同一 Windows 环境、同一组保留依赖版本，使用 `npm run build -- --manifest` 测量。
KB 为十进制；入口大小为 manifest 入口及递归静态依赖的 JS/CSS 总和，
不包含 HTML、字体和路由动态加载的资源，不能等同于完整登录页网络流量。

| 项目 | 精简前 | 精简后 |
| --- | ---: | ---: |
| lockfile 中的包条目（含平台可选依赖） | 770 | 493 |
| `package-lock.json` | 376,046 B | 237,318 B |
| 入口静态 JS/CSS | 611.01 KB | 382.65 KB |
| 入口静态 JS/CSS，逐文件 gzip | 181.09 KB | 108.65 KB |
| 完整前端产物（不含测量 manifest） | 2.041 MB | 2.039 MB |

此次主要减少安装依赖和入口下载/解析量，完整应用功能及相应代码仍保留。
没有重写 Git 历史，也没有用删测试、删数据库迁移或把文档图片改成外链来缩小数字。

## 检查与发布

```powershell
cd frontend
npm run lint
npm run test:i18n
npm run build

cd ../backend
go test ./... -timeout=10m

cd ..
python -m unittest discover -s scripts -p 'test_*.py' -v
```

数据库 CI 继续覆盖 PostgreSQL 15、17。本地未配置 PostgreSQL 时，
相关 Go 集成用例会跳过；Python 的 Docker 验收也是显式启用的独立测试。
本机没有 Docker，因此未执行镜像构建或 PostgreSQL 容器验收。

本次本地验证结果：干净 `npm ci`、前端 lint、多语言检查、类型检查及生产构建通过；
`go test ./... -timeout=10m` 通过；Python 回归测试 23 项通过、Docker 验收 1 项跳过。
另以 `CGO_ENABLED=0` 构建 Go 程序，用临时数据目录启动，验证健康检查 `ok`、
就绪检查 `ready`、`/login` 及其生产 JS/CSS 返回 200，然后关闭临时服务。
Compose 和 GitHub Actions YAML 已解析检查，构建 manifest 的所有模块引用均可解析。

在 Linux Docker 主机的仓库目录执行 `docker compose up -d --build` 可本地构建部署；
现有 Compose 使用 host 网络，Windows 开发请使用上面的 Go/Vite 命令。
GitHub Actions 的 **Build and Release CPA-Helper** 需手动运行，
读取 `VERSION` 发布镜像到 `ghcr.io/i-am-yimin/cpa-helper`，并创建二进制 Release。
同一版本的 Release 不允许重复发布。
