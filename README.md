# Go-Bits-Switch

Go-Bits-Switch 是一个用于管理多个 Go 版本的命令行工具，支持在 Windows 系统上轻松切换不同的 Go 版本。

## 功能特性

- 显示当前 Go 版本信息
- 列出所有已安装的 Go 版本
- 从官方源下载并安装指定版本的 Go
- 在不同 Go 版本之间快速切换
- 自动管理系统环境变量（GOROOT、PATH）
- 支持环境变量的备份和恢复

## 安装

1. 克隆仓库：
```bash
git clone https://github.com/yourusername/go-bits-switch.git
cd go-bits-switch
```

2. 编译安装：
```bash
go build -o go-bits-switch.exe ./cmd
```

3. 将编译后的可执行文件添加到系统 PATH 环境变量中。

## 使用方法

### 显示当前版本
```bash
go-bits-switch -version
```

### 列出已安装版本
```bash
go-bits-switch -list
```

### 安装新版本
```bash
go-bits-switch -install 1.16.5
```

### 切换版本
```bash
go-bits-switch -use 1.16.5
```

### 备份环境变量
```bash
go-bits-switch -backup
```

### 恢复环境变量
```bash
go-bits-switch -restore path/to/backup/file
```

## 注意事项

1. 本工具需要管理员权限来修改系统环境变量
2. 切换版本后需要重启终端或 IDE 以使更改生效
3. 建议在切换版本前先备份环境变量

## 目录结构

```
go-bits-switch/
├── cmd/
│   └── main.go           # 主程序入口
├── internal/
│   ├── config/          # 配置管理
│   ├── download/        # 下载管理
│   ├── env/            # 环境变量管理
│   └── version/        # 版本管理
├── go.mod              # Go 模块文件
└── README.md           # 项目说明文档
```

## 依赖项

- Go 1.16 或更高版本
- golang.org/x/sys

## 许可证

MIT License 