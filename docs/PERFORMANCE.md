# 1.1.0 性能报告

## 2026-09-20：快速缩放的文字排版与纹理优化

Windows 实机反馈：上一轮后仍明显卡顿，快速拖动时进程内存从约 200 MB 涨至 500 MB，停止后明显回落。这与临时分配/缓存峰值相符，但仅凭回落不能确定所有内存的来源。

本轮原生 CPU/分配采样确认：结果列表中带省略号的 `widget.Label` 在宽度改变时通过 `RichText` 重新塑形、计算截断。Fyne 2.8.1 按文字内容缓存纹理，不同截断前缀成为不同缓存项，默认有效期一分钟。现在结果行、表头、正则预览和状态栏使用固定文字、独立省略号及 `container.Clip`，缩放只改变裁剪区域。完整文字供详情、复制及辅助功能使用；省略号前的最后一个字形可能在裁剪边界部分可见。多行提示及可编辑输入框仍使用原控件。

### 测量结果

同一台 Apple M4 Pro、darwin/arm64、Go 1.27.1；修改前为上一轮优化后的已提交代码。测试驱动基准使用 `-benchtime=1s -count=3`，下表为中位数：

| 场景 | 修改前耗时 | 修改后耗时 | 修改前分配 | 修改后分配 |
| --- | ---: | ---: | ---: | ---: |
| 空列表窗口缩放 | 117.3 µs | 32.2 µs | 206,600 B | 36,696 B |
| 一万条结果窗口缩放 | 1,011.1 µs | 38.4 µs | 1,880,515 B | 36,740 B |
| 仅纵向缩放 | 2.305 µs | 2.343 µs | 32 B | 32 B |

原生 OpenGL 探针使用相同的合成一万条结果、600 次 resize 请求、`-interval 0 -settle 2s`，两个独立进程依次运行。没有强制 GC；累计分配来自 `runtime.MemStats.TotalAlloc` 的启动/缩放后差值，包含探针本身开销。进程峰值来自 `/usr/bin/time -l`：

| 指标 | 修改前 | 修改后 |
| --- | ---: | ---: |
| resize 阶段累计 Go 分配 | 1,519.2 MiB | 68.9 MiB |
| resize 阶段 GC 次数 | 27 | 1 |
| 进程峰值 RSS | 276.4 MiB | 251.2 MiB |
| 缩放中每十次请求采样的 Go 堆峰值 | 116.5 MiB | 125.8 MiB |
| resize 序列总时长 | 19.26 s | 19.97 s |

减少分配并不保证每种内存口径都降低：优化后 GC 更少，采样 Go 堆峰值略升，而整个进程峰值 RSS 约降 9%。原生序列时长没有改善，仍受窗口系统、垂直同步和驱动影响，不能把请求次数换算成 Windows FPS，也不能声称已把用户实机的 500 MB 峰值降到某个数值。

回归覆盖：连续 1,180 种宽度保持原文字/纹理几何、省略号边界、完整辅助功能文本、复用行更新、字体样式更新、现有虚拟化和选择逻辑。已执行完整竞态测试、静态检查、Windows amd64 编译，并检查原生 OpenGL 中文截图。Fyne 软件测试画布的嵌套裁剪与原生实现有差异，列表边界以原生截图核验。

### Windows 实机诊断

`ResizePerfProbe.exe` 是独立诊断程序；只构造合成数据，不扫描或复制用户文件，也不更改正式程序的配置。它记录启动、缩放期间、缩放刚结束及空闲后的内存，不强制 GC。Windows JSON 额外包含工作集、进程生命周期峰值工作集及私有内存，API 调用失败时省略这些字段。

```powershell
.\ResizePerfProbe.exe -out perf-empty -rows 0 -iterations 600 -interval 0 -settle 20s
.\ResizePerfProbe.exe -out perf-results -rows 10000 -iterations 600 -interval 0 -settle 20s
```

输出 `startup.json`、`resize-samples.json`、`after-resize.json`、`after-idle.json` 和 CPU/heap profiles。`heap_alloc_bytes` 是未强制 GC 的当前 Go 堆，替代历史探针中的 `heap_live_bytes`，不可直接混比。每十次 resize 请求采样一次，可能遗漏瞬时峰值；`peak_working_set_bytes` 为操作系统记录的全进程峰值。可用 `-capture` 在全部测量后保存原生截图；截图会影响进程最终峰值，因此性能对比时不要启用。

构建探针：`go build -tags perf,migrated_fynedo,no_emoji -o dist/ResizePerfProbe.exe ./cmd/perfprobe`（Windows 上需配置 C 编译器）。正式程序不包含合成数据入口。

## 2026-09-20：窗口缩放增量优化

设置区原先在宽度变化时调用 `grid.Refresh()`。Fyne 的容器刷新会递归刷新全部子控件，导致每次拖动都刷新未变化的文字、图标和固定宽度的右侧面板。现在缩放只更新几何尺寸；显式内容或主题刷新仍刷新子控件并重新测量。宽度未变化的面板复用高度测量，换行改变高度后，结果区在同一轮布局中使用新高度。

保留当前双列、设置区自然高度和结果区随窗口高度伸缩的布局。新增回归测试检查连续缩放不递归刷新面板、不重新测量固定宽度面板，以及换行后的结果区位置；显式刷新仍更新内容。

以下基线为本次修改前的工作区（包含已有的 720×600 默认窗口及设置区布局修改），不是下文的历史 1.1.0 基线。环境为 Apple M4 Pro、darwin/arm64、Go 1.27.1，Fyne 测试驱动，`-benchtime=1s -count=3`，取中位数：

| 指标 | 修改前 | 修改后 |
| --- | ---: | ---: |
| 横向及纵向同时 resize 布局耗时 | 359.6 µs | 116.9 µs |
| 单次分配量 | 557,210 B | 206,600 B |
| 单次分配次数 | 5,586 | 1,924 |
| 仅纵向 resize 布局耗时 | 2.268 µs | 2.281 µs |

复现：`go test -tags ci,migrated_fynedo,no_emoji ./internal/ui -run '^$' -bench 'BenchmarkWindow.*Resize' -benchtime=1s -count=3 -benchmem`。

这些数值衡量 Go 布局工作，不包含 GPU 绘制，也不代表 Windows FPS。弱 GPU Windows 设备仍需分别在空列表、扫描后的结果列表、不同 DPI 下连续拖动窗口边缘验证实际响应。下文为历史记录，其中旧版设置区的滚动/堆叠描述不代表当前布局。

以下数值是 1.1.0 的历史基线。当前 Windows 构建重新使用约 1.8 MiB 的精简字体，常规和粗体样式共用同一资源；macOS 开发预览仍使用 Fyne 默认字体。下表来自 macOS，不能作为 Windows FPS 或任务管理器内存的保证。

本轮针对 Windows 用户报告的“刚启动就占用约 300 MB，拖拽 / resize 卡顿，部分区域不能跟随缩放”。程序已更新并交叉编译；以下实测来自 **Apple M4 Pro、darwin/arm64、Go 1.27.1**，不能作为 Windows FPS 或任务管理器内存的保证。

## 定位结果

- 原主题在正常、粗体等字体样式中解析完整 16 MB CJK 字库。原生界面启动堆采样的大多数存活内存来自字体表及字形解析。仅改为完整 TrueType 字库没有改善，故没有采用该方案。
- 原结果表格固定六列宽度，而且每个单元格都创建 Checkbox、Label 和 Stack，文字未变也调用 `SetText`，产生额外刷新。
- 原发布未设置 `migrated_fynedo`。应用后台更新虽已使用 `fyne.Do`，仍进入 Fyne 兼容线程检查；新版构建明确启用已迁移模式。
- Fyne 2.8.1 提供重绘、渲染列表及对象遍历改进，已升级并锁定版本。[上游发布说明](https://github.com/fyne-io/fyne/releases/tag/v2.8.1)
- 预览计划重复复制文件元数据，复制流程为每个文件分配 256 KiB 缓冲，且 `io.CopyBuffer` 的 `ReaderFrom` 快速路径可能绕过该缓冲。

## 本机对比

原生测试为两个独立进程，均使用空源目录、默认中文设置、同一套 `cmd/perfprobe`。显示界面后空闲两秒，采样并执行 120 次窗口 resize，再次采样。存活堆在显式 GC 后测量；峰值 RSS 来自 `/usr/bin/time -l`，包含图形驱动及剖析器自身开销。系统缓存和调度会影响数值。

| 指标 | 原版 | 1.1.0 |
| --- | ---: | ---: |
| 启动 Go 存活堆 | 62.0 MiB | 12.9 MiB |
| 120 次 resize 后 Go 存活堆 | 62.0 MiB | 12.9 MiB |
| 整个测试进程峰值 RSS | 238.0 MiB | 168.7 MiB |
| resize 阶段累计 Go 分配 | 41.2 MiB | 35.0 MiB |

软件测试驱动基准（相同测试代码、2 秒采样、不含 GPU 绘制）：

| 指标 | 原版 | 1.1.0 |
| --- | ---: | ---: |
| 单次窗口布局 resize | 261 µs | 132 µs |
| 单次 resize 分配次数 | 2,598 | 1,513 |
| 10,000 条结果切换单个勾选并重算计划 | 6.56 ms | 2.28 ms |
| 单次重算分配量 | 11.42 MiB | 2.00 MiB |

原生窗口操作仍受操作系统事件、显示器刷新和驱动影响，因此没有把探针的 resize 调度时长换算成 FPS。没有用强制清空工作集、持续强制 GC 或硬内存上限制造低内存读数。

## 实现与回归保护

- 默认字体精简为约 1.8 MiB，包含 7,670 个字符，覆盖 GB2312、Latin 与全部界面文案。生僻字、额外文字及 Emoji 使用系统字体回退；不保证没有安装相应系统字体时的额外字符覆盖。
- 字体测试验证全部中英文文案有字形，并阻止误将大于 3 MiB 的完整字库重新设为默认字体。
- 表格替换为虚拟行列表，每行仅一个复选框和五个标签；可见行复用，文本未变时不刷新。列宽由可用宽度计算。
- 设置区在窄窗口堆叠，通过独立滚动保留所有控件；结果区保留空间。可拖动中间分隔条调整高度。
- 缩放测试覆盖 800、1000、1280、1600 宽度和 10,000 条结果，检查列宽伸缩、面板边界及虚拟化；检查回收行的勾选不会修改其他文件。
- 计划生成复用选择缓冲、目标索引，只保存每组计数；不重复保存组内所有文件副本。保持原有排序、分组及冲突语义。
- 复制缓冲改为每批复用；预检可取消；进度更新有背压；关闭后不会再打开任务结果对话框。

## 复现

```sh
go test -race -tags ci,migrated_fynedo,no_emoji ./...
go vet -tags ci,migrated_fynedo,no_emoji ./...
go test -tags ci,migrated_fynedo,no_emoji ./internal/ui -run '^$' \
  -bench 'BenchmarkWindowResize|BenchmarkPreviewSelection' -benchtime=2s -benchmem
```

真实桌面探针（会打开窗口、自动 resize 并退出；不读取用户数据）：

```sh
go build -tags perf,migrated_fynedo,no_emoji -o dist/perfprobe ./cmd/perfprobe
./dist/perfprobe -out dist/perf
go tool pprof -top dist/perf/resize-cpu.pprof
go tool pprof -top -inuse_space dist/perf/startup-heap.pprof
```

Windows 上将输出名称改为 `dist/perfprobe.exe`，在配好 C 编译器的终端运行。探针生成 JSON、CPU 和 heap profile；原始文件中的 `heap_live_bytes` 是 Go 堆，不能直接与任务管理器的整个进程内存比较。

## Windows 验收边界

本机已通过测试、竞态检测、静态检查、窗口布局渲染及 Windows amd64 交叉编译。新 exe 的 Windows 实机响应、显卡驱动、100% / 150% / 200% DPI 和远程桌面场景仍需复测。

建议先确认文件属性版本为 **1.1.0**，刚启动时观察内存，然后连续拖动 / 缩放窗口、检查窄窗口下滚动设置区，再扫描较大目录比较滚动与勾选响应。若仅 Windows 仍卡顿，需结合原生探针的 CPU profile 和该机器的显示环境继续定位，不能用本机测试结论替代实机结果。
