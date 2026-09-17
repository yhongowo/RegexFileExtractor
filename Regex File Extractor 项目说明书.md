# Regex File Extractor 项目说明书

## 1. 项目概述

项目名称：**Regex File Extractor**

这是一个面向 Windows 平台的轻量级桌面 GUI 工具，用于根据用户配置的多个正则表达式规则，从指定目录中筛选并提取符合条件的文件。

程序应尽可能简单、可靠、易维护。

主要使用场景：

用户有一个包含大量测试数据文件的目录，例如：

```text
Spectrum_Data_X01Y01.csv
Spectrum_Data_X01Y02.csv
X01Y01.csv
X02Y02.csv
Log_20260916.txt
```

用户可以保存多个 Regex 规则

程序扫描指定目录后：

- 找出符合规则的文件
- 显示匹配结果
- 允许用户预览
- 将符合条件的文件复制到目标目录（不需要move的功能）

该程序主要用于工程测试数据、CSV 文件、日志文件等批量文件筛选场景。

---

# 2. 技术选型

必须使用以下技术栈：

```text
Language: Go
GUI: Fyne
Config: JSON
Target OS: Windows 10 / Windows 11
Architecture: amd64
```

Go 版本建议：

```text
Go >= 1.24
```

Fyne：

```text
fyne.io/fyne/v2
```

项目应尽量减少第三方依赖。

除 Fyne 外，优先使用 Go 标准库。

例如：

```text
regexp
os
io
path/filepath
encoding/json
sync
```

---

# 3. 核心设计原则

项目需要遵循以下原则：

1. 简单，稳定，易维护
4. UI 清晰，不过度设计
6. 业务逻辑与 GUI 分离
7. Windows 用户无需安装额外 Runtime
8. 最终能够构建成单个 `.exe`

---

# 5. 配置文件存储

Windows 推荐保存到：

```text
%APPDATA%\RegexFileExtractor\config.json
```

---

# 7. Regex 规则

用户可以添加，删除，修改，启用禁用，为 Regex 设置名称

保存前需要执行：

```go
regexp.Compile(pattern)
```

如果 Regex 不合法，禁止保存。

# 8. GUI 总体设计

语言：中英文可切换

整体设计风格：

```text
Modern
Minimal
Windows 11 inspired
```

设计关键词：

```text
clean
minimal
spacing
cards
modern desktop utility
```
