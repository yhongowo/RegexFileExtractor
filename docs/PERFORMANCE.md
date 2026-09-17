# 1.1.0 性能报告

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
