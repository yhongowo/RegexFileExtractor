# Regex File Extractor

Windows 10 / 11（amd64）桌面文件提取工具，使用 Go + Fyne。可保存多条正则规则，每次只选中一条，递归筛选文件名，预览后复制到目标目录。

## 直接使用

运行 `dist/RegexFileExtractor.exe`。只需此单个文件，不需要安装 Go、Fyne、.NET 或额外的编译器运行库。界面使用 Fyne 默认字体，中文字符由系统字体回退显示。Fyne 界面使用系统 OpenGL，需要正常工作的显卡驱动。

1. 选择源目录和目标目录。目标目录也可以手动输入，复制时自动创建。
2. 选择一条规则，或添加 / 编辑自己的规则。规则编辑器可输入示例文件名即时测试。
3. 选择 `auto`、`group` 或 `flat`，已有目标文件默认跳过，可切换为覆盖。
4. 点击 **扫描文件**。程序始终递归扫描，无需设置递归选项。
5. 查看文件名、完整源路径、大小、命中规则和**目标相对路径**，勾选需要的文件。点击文字单元格可以查看完整字段并复制详情。
6. 点击 **复制选中文件**。覆盖模式会在操作前显示确认。扫描、复制均可取消。

源文件始终保留；程序不提供移动功能或文件内容预览。复制保留文件内容与修改时间，不复制 ACL、扩展属性或备用数据流。任务详情最多保留前 100 条具体错误，但错误总数完整统计。

右上角可随时切换中英文（任务执行期间设置锁定）。源目录、目标目录、所选规则、布局及冲突策略会记住。修改源目录、目标目录或规则后，需要重新扫描，避免复制旧的结果。

## 精确匹配

匹配对象是**完整文件名，包含扩展名**，不包含目录路径。内部强制完整匹配，不会将任意子串命中视为成功。

```regex
X\d+Y\d+\.csv
```

匹配 `X01Y01.csv`，不匹配 `Spectrum_Data_X01Y01.csv`、`X01Y01.csv.bak` 或 `X01Y01.CSV`。

默认区分大小写。若要忽略大小写：

```regex
(?i)X\d+Y\d+\.csv
```

允许可选前缀：

```regex
(?:Spectrum_Data_)?X\d+Y\d+\.csv
```

使用 Go 标准库 RE2 语法，不支持前后向断言和反向引用。非法表达式、空名称、重复名称均不能保存。规则列表可为空，但扫描前必须选中一条规则；不存在多选或多个规则同时生效。

## 三种组织方式

例如源目录有：

```text
A/X01Y01.csv
B/X01Y01.csv
B/X02Y02.csv
```

**flat**：全部平铺；同名文件按照冲突策略处理。

```text
目标目录/
  X01Y01.csv
  X02Y02.csv
```

**group**：每个完整文件名一组，文件夹采用去掉最后一个扩展名的名称。即使组内只有一个文件，也建文件夹并编号。

```text
目标目录/
  X01Y01/
    X01Y01_001.csv
    X01Y01_002.csv
  X02Y02/
    X02Y02_001.csv
```

**auto**（默认）：同名文件分组，唯一文件平铺。

```text
目标目录/
  X01Y01/
    X01Y01_001.csv
    X01Y01_002.csv
  X02Y02.csv
```

分组规则：

- “同名”按完整文件名含扩展名判断，忽略大小写，以符合 Windows 文件系统常规行为。这与正则是否区分大小写互相独立。
- 按已勾选的文件重新计算分组。取消某些勾选可能使 auto 从分组变为平铺，或改变序号；目标路径列会同步更新。
- 序号按完整源路径的 Go 字符串顺序排列，至少三位，超过 999 时自然扩展。不保留原有目录层级。
- `X.csv` 与 `X.txt` 是不同组，但文件夹名都想使用 `X`，此时自动使用 `X`、`X_002`。auto 中若存在名为 `X` 的平铺文件，也会避免文件与文件夹占用同一路径。
- 编号基于本次选择重新生成；不从目标目录的现有编号继续累加。重复提取会对这些确定的目标路径应用跳过 / 覆盖策略。

## 冲突与取消

- **跳过**：不更改已存在的同名目标文件。flat 模式中的源文件同名冲突保留第一个成功复制的文件。
- **覆盖**：完整写入并同步临时文件后，再替换目标；写入失败或取消不会先截断旧目标。flat 模式同名源文件保留最后一个成功复制的版本。
- 不覆盖源文件，也不通过目标子目录中的符号链接写入其他位置。源目录与目标目录不能相同。
- 目标目录位于源目录内部时，扫描会排除整个目标子树，避免重复扫描已提取的内容。源目录内的符号链接不跟随。
- 复制前检查源文件大小和修改时间；若文件自扫描后已改变，报告该文件失败并提示重新扫描。复制期间也会检查变化。
- 取消扫描丢弃部分扫描结果；取消复制保留已完成的文件并清理当前未完成文件。不提供批量回滚。
- 无权限或读取失败会报告详情，其他文件继续处理。目标同名目录不作为可覆盖文件处理。

## 配置

Windows：`%APPDATA%\RegexFileExtractor\config.json`。

本地 macOS 开发时为 `~/Library/Application Support/RegexFileExtractor/config.json`，Linux 为系统用户配置目录下的 `RegexFileExtractor/config.json`。

配置以 JSON 保存，先校验再写临时文件并替换。配置损坏时原文件保留，程序使用默认设置并暂停保存；可选择“备份并重置”，生成带时间戳的原始配置备份。若先取消，可从“关于”重新进入恢复对话框。

## 开发与测试

要求 Go **1.24+**、Fyne **2.8.1**，桌面编译需 C 编译器及对应平台的图形开发环境。应用业务逻辑仅依赖 Go 标准库；其余模块是 Fyne 的依赖。

```sh
go mod download
go test -race -tags ci,migrated_fynedo,no_emoji ./...
go vet -tags ci,migrated_fynedo,no_emoji ./...
go run -tags migrated_fynedo,no_emoji ./cmd/regexfileextractor
```

`ci` 标签使用无窗口驱动运行界面测试，不用于发布。核心测试涵盖完整匹配、递归扫描、输出目录排除、三种布局、大小写与目录命名冲突、跳过 / 覆盖、取消、源文件变化保护和配置恢复；UI 测试验证单选、选择集重算、旧结果失效、忙碌状态及语言切换。

可选的中英文界面截图导出：

```sh
RFE_SCREENSHOTS="$PWD/dist" go test -tags ci,migrated_fynedo,no_emoji ./internal/ui -run TestRenderScreenshots -count=1
```

截图由 Fyne 软件测试画布生成，用于布局检查，非 Windows 实机截图。默认窗口及 800、1600 像素宽度均会导出。窄窗口下设置区自动改为上下排列，可滚动；中间分隔条可调整设置区和结果区的高度。

## 构建 Windows 单文件

Macos 编译
- to macos apple silicon
```bash
go build -v -o dist/RegexFileExtractor ./cmd/regexfileextractor
```
- to windows
```bash
GOCACHE=/tmp/regexfileextractor-go-build GOMODCACHE=/tmp/regexfileextractor-go-mod CC=/tmp/llvm-mingw-20260908-ucrt-macos-universal/bin/x86_64-w64-mingw32-clang WINDRES=/tmp/llvm-mingw-20260908-ucrt-macos-universal/bin/x86_64-w64-mingw32-windres sh scripts/build-windows.sh
```

Windows 开发机安装 Go 与 [MSYS2](https://www.msys2.org/)，在 UCRT64 终端安装工具链：

```sh
pacman -S --needed mingw-w64-ucrt-x86_64-gcc
```

将 `C:\msys64\ucrt64\bin` 加入当前 PowerShell 的 PATH，然后在项目目录执行：

```powershell
./scripts/build-windows.ps1
```

输出 `dist/RegexFileExtractor.exe` 及 `dist/SHA256SUMS.txt`。构建脚本嵌入 Windows 版本信息、普通用户权限、高 DPI 和长路径声明，并隐藏控制台窗口。

macOS / Linux 可使用 [LLVM-MinGW](https://github.com/mstorsjo/llvm-mingw) 或相应 MinGW 工具链交叉编译：

```sh
CC=/path/to/bin/x86_64-w64-mingw32-clang \
WINDRES=/path/to/bin/x86_64-w64-mingw32-windres \
sh scripts/build-windows.sh
```

仓库提供 `.github/workflows/windows.yml`，可在 GitHub 的 Windows runner 上测试并构建下载产物。当前本地生成的 exe 为 macOS 交叉编译产物；Windows 10 / 11 实机启动、高 DPI、网络盘及权限场景仍需在 Windows 上验收。

## 项目结构

```text
cmd/regexfileextractor/  GUI 入口
internal/core/          匹配、递归扫描、布局计划、文件复制
internal/config/        JSON 配置、校验与恢复
internal/ui/            Fyne 界面、中英文、主题
assets/                资源及旧版字体文件
scripts/               Windows 构建脚本、资源及 manifest
```

界面使用 Fyne 默认字体，中文字符通过系统字体回退显示，效果取决于操作系统安装的字体。`assets/RFESans-Regular.otf` 保留为旧版资源，不再嵌入程序。

## 1.1.0 性能优化

- 升级到 Fyne 2.8.1，发布时启用 `migrated_fynedo`；后台 UI 更新均通过 `fyne.Do` / `fyne.DoAndWait`。标志随 exe 生效，不依赖旁置配置文件。
- 旧版使用精简的 CJK 字体，以避免完整字库的启动解析开销；当前界面已改用 Fyne 默认字体。
- 结果区改为虚拟列表，仅复用可见行；列宽随窗口改变，未变文字不重复刷新。
- 窄窗口下设置区自适应排列并滚动；布局不反复收缩、拉伸同一面板。
- 预览计划按组保存计数，避免重复复制每个文件的元数据；复用选择缓冲和目标索引。
- 每批复制复用一块 256 KiB 缓冲；进度回调施加背压，避免拖拽时累积 UI 更新队列。

测量数据、复现命令及 Windows 验收边界见 [性能报告](docs/PERFORMANCE.md)。

旧版字体资源可用 `fonttools==4.60.2` 重新生成：

```sh
python scripts/subset-font.py /path/to/NotoSansCJKsc-Regular.otf
```

这是开发期工具；当前构建不加载此字体，也不需要 Python。
