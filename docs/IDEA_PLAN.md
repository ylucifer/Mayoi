# IDEA 化改造策略与实施指南（路线 A）

> 本文档是 Mayoi → IDEA 风格 fork 的**长期指导文档**。每次动手改造前先读一遍，
> 每次合并上游后如有新的经验，回来更新它。本文件属于我们自己的文档，不需要推给上游。

## 1. 背景与目标

- 上游：`github.com/egoist/Mayoi`，VS Code 风格的原生 Go 桌面编辑器，UI 由
  `github.com/egoist/mygo` 框架绘制。
- 我们：`github.com/ylucifer/Mayoi`，目标是把界面改成 **IDEA 风格**（新 UI 布局、
  快捷键、语法高亮配色、主题色），同时**保持从上游合并新功能的能力**。
- 我们的代码**不推送上游**；上游的更新只通过 `git merge` 进来。

## 2. 现状评估结论（2026-11 评估）

### 2.1 架构事实

| 事实 | 对我们的影响 |
|---|---|
| 上游是单体应用：`module mayoi`，全部 UI/逻辑在 `cmd/mayoi` 一个 main 包（244 个 Go 文件，约 10.8 万行） | 不存在可单独引用的组件层 |
| 可复用部分在 `internal/`（settings、keymap、highlight、git、vfs、workspace、remote 等 15 个包） | Go 规定跨模块不能 import 别人的 `internal/`，“上游当依赖”路线被语言层面挡死 |
| `app` 是 200+ 字段的 god object；UI 组装集中在 `view.go`（约 1800 行） | 布局改动必然落在 `view.go`，这是唯一绕不开的冲突点 |
| `host` 接口隔离了原生桌面（对话框/剪贴板/回收站），测试可无头运行 | 平台相关行为有现成的注入缝 |
| `keybindings.json`、`settings.json` 支持热加载；主题是数据定义；高亮基于 chroma | 快捷键与配色有“零代码改动”通道 |
| 测试 3.9 万行 / 140 个文件（占比 36%），含 `MAYOI_SNAPSHOT` 截图快照机制 | 改坏能被测试兜住；界面可自动截图比对 |

### 2.2 代码质量结论

工程纪律 9/10（测试、文档 `docs/ARCHITECTURE.md`、注释、并发处理都属专家级），
模块化 5/10（巨石单包、god object、无组件边界）。**综合 8/10，适合做“换皮型”扩展，
不适合做插件生态或架构级重构。** 投入值得。

### 2.3 已完成的第一步

提交 `Lay out the window as IDEA's new UI`（`idea-layout-dev` 的起点）已落地
IDEA 新 UI 布局：主工具栏、左右工具窗条、源码管理停靠右侧、30pt 状态栏、
44pt 工具条等。该提交已按第 4 节重构为贴片式：实现住在 `idea_view.go`
（布局）与 `idea_metrics.go`（度量）这两个我们独占的新文件里，`view.go` 与
`sidebar.go` 的上游函数体原样保留，只在头部插 `if ideaLayout` 前置分支交出
工作——这正是路线 A 的雏形。

## 3. 路线选型记录

| 路线 | 内容 | 结论 |
|---|---|---|
| **A：贴片式改造 + 数据驱动隔离** | 核心不动，改动收敛为“插入式贴片”+ 数据文件 | **采用** |
| B：库化拆分 + 自建入口 | 把 `cmd/mayoi` 拆成可 import 的包，写我们自己的 main | 否决：`internal/` 无法跨模块引用；拆分是与上游差异最大的改法，每次上游合并都变成“持续重新实现”，除非上游接受拆包 PR |
| C：低频追上游，直接改 | 拉长合并周期，不管冲突 | 作为兜底认识即可，不主动采用；路线 A 在此节奏下同样成立 |

## 4. 路线 A 核心原则

### 4.1 贴片式改造

**能用“新增文件”解决的就不改旧文件；必须改旧文件的，只插不改写。**

- ❌ 改写式：重写上游函数体 → 上游任何一行变动都冲突。
- ✅ 贴片式：上游函数原样保留，入口处插前置分支：

```go
func (a *app) sidebar(c *ui.Context) {
    if ideaLayout {
        a.ideaSidebar(c)   // 实现放在我们自己的新文件里，零冲突
        return
    }
    // 上游原有代码，一个字不动
}
```

- 我们自己的实现一律住在**新增文件**（`idea_*.go`）里，上游永远不会碰它们。
- 每个布局改动保持**一个语义化提交**（72 字符内祈使句主题，细节放 body，遵循 AGENTS.md），
  rebase / 冲突处理时粒度清晰。

### 4.2 数据驱动隔离

优先利用上游现成的配置通道，让改动成为“加数据”而不是“改代码”：

| 要改的 | 做法 | 冲突风险 |
|---|---|---|
| 快捷键 | 提供一份 IDEA 风格 `keybindings.json`（VS Code 格式，`internal/keymap` 原生支持热加载）；如需改默认值，只动 `internal/keymap` 单一位置 | ≈0 |
| 语法高亮配色 | 在 `internal/highlight`（chroma）新增 IDEA 配色主题文件 + 一行注册 | ≈0 |
| UI 颜色 / 度量常量 | 新增 `idea_theme.go`、`idea_metrics.go`，集中放 IDEA 色板与度量（状态栏 30pt、工具条 44pt 等），`view.go` 只引用 | ≈0 |

“隔离”的含义：这些文件是我们独占的，上游永远不改，合并时天然免疫。

## 5. 待办的 Git 基础工作（优先级最高）

当前仓库是 shallow clone（`.git/shallow` 存在，本地仅 2 个提交），**不处理则无法正常
merge 上游**。执行一次：

```powershell
git fetch --unshallow
git remote add upstream https://github.com/egoist/Mayoi.git
git fetch upstream
```

分支策略：

- `main`：纯净跟踪上游（我们的分支从它分出，永不直接提交）。
- `idea-layout-dev`（或后续改名 `idea`）：只放我们自己的提交，定期 merge 上游。
- 远程 `origin` 只推我们的分支，不推 `main`（我们的代码不回上游）。

## 6. 分项改造指南

### 6.1 布局（已有基础，继续收敛）

- 已完成：布局提交（`Lay out the window as IDEA's new UI`）的主工具栏 +
  工具窗条 + 右侧源码管理，并已收敛为 `idea_view.go` / `idea_metrics.go` 贴片。
- 后续收敛动作：
  1. （已做）`view.go` 里的 IDEA 度量常量（30pt 状态栏、44pt 工具条、圆角按钮等）
     已抽到 `idea_metrics.go`；上游常量（`titleBarH` 等在 `view.go` 顶部）保持原值，
     上游文件通过 `titleH()`、`statusH()`、`statusItemH()`、`statusTextSize()` 等
     引用函数取值——引用式小改，每处一行，冲突易解。
  2. IDEA 布局的颜色引用统一走 `idea_theme.go`。
  3. `view.go` 内的改动保持为“结构选择 + 引用”，每次 merge 后检查贴片是否仍在前置位置。
- 改上游函数时的检查：是否只在前部插了 `if ideaLayout {...}`？函数体其余部分是否原样？

### 6.2 快捷键（优先做，成本最低）

1. 梳理 IDEA 默认快捷键（对照 JetBrains 官方 keymap 文档）与上游命令 id 的映射。
2. 产出 `keybindings-idea.json`（VS Code 格式），随应用分发或作为预设；
   `internal/keymap` 已支持热加载，代码零改动。
3. 注意上游 keymap 支持和弦（chord）与 VS Code 命令名别名，IDEA 常用键
   （如 `Ctrl+E` recent files、`Ctrl+Shift+N`、`Alt+Enter` 等）大多能直接映射，
   上游没有的命令先记录在本文档附录，评估补命令的贴片点。

### 6.3 语法高亮配色

- `internal/highlight` 基于 chroma（含 `vue.xml` 等自定义定义）。
- 新增 IDEA 配色主题：只加文件 + 注册一行，不改上游主题逻辑。

### 6.4 UI 主题与外观

- 上游主题集中在 `cmd/mayoi/theme.go`（约 70KB）。原则：**不改上游主题定义**，
  IDEA 色板放 `idea_theme.go`，通过 `ideaLayout` 分支选择色板来源。
- 明暗两套都要做，注意 `themeSource()` 的明暗判断分支保持上游原样。

### 6.5 其他后续项（暂缓，做前在此补计划）

- 图标体系（IDEA 的工具窗图标、项目图标）。
- 欢迎页 / 设置窗口的 IDEA 化。
- 启动画面、关于窗口。

## 7. 合并上游的标准流程

```powershell
git fetch upstream
git checkout idea-layout-dev
git merge upstream/main
# 解决冲突（预期冲突面：view.go / sidebar.go 的贴片行）
go build ./... 
go test ./cmd/mayoi/ -run TestLayout   # 不跑全量测试（AGENTS.md：全量慢）
# 界面验证：MAYOI_SNAPSHOT=out.png go run ./cmd/mayoi  生成截图比对
```

冲突处理原则：

1. 优先保上游语义，把我们的贴片“重新贴上去”，而不是把上游代码改成我们的版本。
2. 上游重写了某个函数导致贴片失效：不要硬合并，重新读函数，找新的插入点重贴。
3. 冲突经验写回本文档第 9 节，逐步积累成冲突地图。

## 8. 每次改动前的检查清单

- [ ] 这次改动能否用“新增 `idea_*.go` 文件”实现？
- [ ] 若必须改上游文件：是否只插不改写？是否用 `if ideaLayout` 前置分支？
- [ ] 度量常量、颜色是否进了 `idea_metrics.go` / `idea_theme.go`，而不是散在 view.go？
- [ ] 是否一个语义化提交？
- [ ] 快捷键/配色类的改动，是否走了数据通道（json/主题文件）而非代码？
- [ ] 测试是否覆盖（至少跑相关包的测试 + 截图快照）？

## 9. 合并经验记录（持续追加）

> 每次合并上游后，在此记录：日期、上游带来的冲突文件、解决方式、贴片是否需要重贴。

- （暂无）

## 10. 附录

- 上游架构导读：`docs/ARCHITECTURE.md`（逐区域走读源码，读懂上游必读）。
- 功能清单：`docs/FEATURES.md`；设置项：`docs/SETTINGS.md`。
- 评估原始结论日期：2026-11-03，基于布局提交（`Lay out the window as IDEA's new UI`）。
