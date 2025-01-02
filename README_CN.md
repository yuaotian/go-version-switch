# go-version-switch

<div align="center">

[![Release](https://img.shields.io/github/v/release/yuaotian/go-version-switch?style=flat-square&logo=github&color=blue)](https://github.com/yuaotian/go-version-switch/releases/latest)
[![Go Version](https://img.shields.io/badge/go-%3E%3D%201.16-blue)](https://img.shields.io/badge/go-%3E%3D%201.16-blue)
[![Release Build](https://github.com/yuaotian/go-version-switch/actions/workflows/release.yml/badge.svg)](https://github.com/yuaotian/go-version-switch/actions/workflows/release.yml)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

🔄 一个简单的 Go 版本管理工具，专为 Windows 系统打造

[English](./README.md) | 简体中文

</div>

## ✨ 特性

- 🔍 实时显示当前 Go 版本信息
- 📋 管理多个已安装的 Go 版本
- ⬇️ 自动下载安装官方发布版本
- 🔄 快速切换不同 Go 版本
- ⚙️ 智能管理系统环境变量
- 💾 支持环境配置备份恢复
- 🔒 安全的环境变量回滚机制
- 🌐 支持多架构（x86/x64/arm/arm64）

## 🚀 快速开始

### 📥 安装方式

#### 方法 1：直接下载

从 [Releases](https://github.com/yuaotian/go-version-switch/releases) 页面下载最新版本。

#### 方法 2：从源码编译

```bash
# 克隆仓库
git clone https://github.com/yuaotian/go-version-switch.git
cd go-version-switch

# 编译
go build -v -o bin/go-version-switch.exe ./cmd/main.go 

# 测试
./bin/go-version-switch -install 1.23.4 -arch x86

#编译+测试
go build -v -o bin/go-version-switch.exe ./cmd/main.go && ./bin/go-version-switch -install 1.23.4 -arch x86


# 将可执行文件添加到 PATH 环境变量
# 建议将编译后的文件复制到 C:\Program Files\go-version-switch\ 目录下
# 或者使用命令一键添加到PATH：
 setx /M PATH "%PATH%;C:\Program Files\go-version-switch"
```

### 🎯 基础使用

```bash
# 查看帮助信息
go-version-switch -h

# 查看当前版本
go-version-switch -version

# 列出所有已安装版本或更新版本列表
go-version-switch -list

# 列出所有版本之前强制更新版本列表
go-version-switch -list -update

# 安装特定版本
go-version-switch -install 1.23.4

# 安装特定版本和架构
go-version-switch -install 1.23.4 -arch x64

# 切换到本地已安装版本
go-version-switch -use 1.23.4

# 回滚环境变量配置
go-version-switch -rollback
```

## 📁 项目结构

```md
go-version-switch/
├── 📂 cmd/
│   └── main.go                 # 程序入口
├── 📂 internal/
│   ├── config/                # 配置管理
│   │   └── config.go         # 配置处理
│   └── version/              # 版本管理
│       ├── common.go        # 通用函数
│       ├── download.go      # 下载功能
│       ├── env.go          # 环境变量处理
│       ├── goversion.go    # 版本信息
│       ├── install.go      # 安装逻辑
│       ├── list.go        # 版本列表
│       ├── releases.go    # 发布管理
│       └── version.go     # 版本控制
├── 📂 data/               # 运行时数据
│   └── config/            # 配置文件
├── 📄 go.mod              # 依赖管理
├── 📄 go.sum              # 依赖校验
└── 📝 README.md           # 项目文档
```

## ⚙️ 系统要求

- Windows 10/11
- Go 1.16+（仅编译时需要）
- 管理员权限（用于修改环境变量）
- 稳定的网络连接（下载新版本时需要）

## 🔧 故障排除

### 常见问题

1. **权限不足**
   ```bash
   错误：需要管理员权限
   解决：以管理员身份运行命令提示符
   ```

2. **下载失败**
   ```bash
   错误：下载超时
   解决：检查网络连接或使用代理
   ```

3. **版本切换失败**
   ```bash
   错误：环境变量更新失败
   解决：使用 -rollback 命令恢复之前的配置
   ```

## 👨‍💻 开发者指南

### 构建项目

```bash
# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建和测试
go build -v -o bin/go-version-switch.exe ./cmd/main.go && ./bin/go-version-switch -install 1.23.4 -arch x86
```

### 代码贡献

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 📌 注意事项

1. 🔐 需要管理员权限来修改系统环境变量
2. 🔄 切换版本后需要重启终端或 IDE
3. 💾 定期备份环境变量配置
4. ⚠️ 确保网络连接稳定
5. 📦 不要手动修改工具的数据目录

## 🤝 贡献指南

- 提交 Issue 前请先搜索是否已存在类似问题
- Pull Request 请提供详细的描述
- 遵循项目的代码规范
- 确保提交的代码已经过测试

## 📄 开源协议

本项目采用 [MIT](./LICENSE) 开源协议。 