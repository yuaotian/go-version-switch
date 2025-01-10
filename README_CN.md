# go-version-switch

<div align="center">

[![Release](https://img.shields.io/github/v/release/yuaotian/go-version-switch?style=flat-square&logo=github&color=blue)](https://github.com/yuaotian/go-version-switch/releases/latest)
[![Go Version](https://img.shields.io/badge/go-%3E%3D%201.16-blue)](https://img.shields.io/badge/go-%3E%3D%201.16-blue)
[![Release Build](https://github.com/yuaotian/go-version-switch/actions/workflows/release.yml/badge.svg)](https://github.com/yuaotian/go-version-switch/actions/workflows/release.yml)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

🔄 功能强大的 Go 版本管理工具，专为 Windows 系统打造

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
- 📦 本地安装包检测和使用
- 🔧 自动目录完整性验证
- 🔄 智能架构切换与本地包检测

## 🚀 快速开始

### 📥 安装方式

#### 方法 1：直接下载

1. 从 [Releases](https://github.com/yuaotian/go-version-switch/releases) 页面下载最新版本
2. 解压到指定目录（推荐：C:\Program Files\go-version-switch\）
3. 添加到 PATH 环境变量：
   ```powershell
   # 添加到系统环境变量并重启终端
   setx /M PATH "%PATH%;C:\Program Files\go-version-switch"
   # 安装指定版本
   govs.exe -install 1.23.4 -arch x64
   # 切换到已安装版本
   govs.exe -use 1.23.4
   # 切换架构
   govs.exe -arch x64
   # 回滚环境变量
   govs.exe -rollback
   # 列出所有可用版本
   govs.exe -list
   # 强制更新版本列表
   govs.exe -list -update
   # 查看帮助信息
   govs.exe -help

   ```

#### 方法 2：从源码编译

```bash
# 克隆仓库
git clone https://github.com/yuaotian/go-version-switch.git
cd go-version-switch

# 编译
go build -o bin/govs.exe ./cmd

# 测试安装
./bin/go-version-switch -install 1.23.4 -arch x64

# 一键编译和测试
go build -v -o bin/govs.exe ./cmd/main.go && ./bin/govs.exe -install 1.23.4 -arch x64
```

### 🎯 基础使用

```bash
# 查看帮助信息
go-version-switch -h

# 列出所有可用版本
go-version-switch -list

# 强制更新版本列表
go-version-switch -list -update

# 安装指定版本
go-version-switch -install 1.23.4 -arch x64

# 切换到已安装版本
go-version-switch -use 1.23.4

# 直接切换架构
go-version-switch -arch x64
go-version-switch -arch x86

# 回滚环境变量
go-version-switch -rollback
```

### 🔧 高级功能

#### 架构管理
```bash
# 支持的架构列表
x86, 386, 32       (32位)
x64, amd64, x86-64 (64位)
arm                (ARM)
arm64              (ARM64)

# 带本地包检测的架构切换
go-version-switch -arch x64  # 自动查找并使用本地安装包
```

#### 本地包支持
- 自动检测 down/ 目录中的安装包
- 优先使用本地安装包
- 安装前验证包完整性

#### 环境变量管理
- 修改前自动备份
- 安全的回滚机制
- 智能 PATH 管理
- GOROOT 和 GOARCH 处理

## 📁 项目结构

```
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
│       ├── install.go      # 安装逻辑
│       ├── list.go        # 版本列表
│       ├── releases.go    # 发布管理
│       └── version.go     # 版本控制
├── 📂 bin/
│   └── data/              # 运行时数据
│       ├── go-version/   # Go 安装目录
│       ├── down/         # 下载缓存
│       ├── backup_env/   # 环境变量备份
│       └── config/       # 配置文件
├── 📄 go.mod              # 依赖管理
└── 📝 README.md           # 项目文档
```

## ⚙️ 系统要求

- Windows 10/11
- Go 1.16+（仅编译时需要）
- 管理员权限
- 网络连接（用于下载）

## 🔧 故障排除

### 常见问题

1. **权限错误**
   ```bash
   错误：需要管理员权限
   解决：以管理员身份运行
   ```

2. **下载问题**
   ```bash
   错误：下载失败
   解决：
   - 检查网络连接
   - 使用 down/ 目录中的本地包
   - 检查代理设置
   ```

3. **版本切换失败**
   ```bash
   错误：环境变量更新失败
   解决：
   1. 使用 -rollback 恢复
   2. 检查文件权限
   3. 关闭所有 Go 进程
   ```

4. **目录完整性**
   ```bash
   错误：Go 安装不完整
   解决：
   - 工具将自动尝试使用本地包修复
   - 检查 down/ 目录中的安装包
   ```

## 👨‍💻 开发指南

### 构建

```bash
# 安装依赖
go mod download

# 运行测试
go test ./...

# 带版本信息构建
go build -ldflags="-X 'main.Version=v1.0.0'" -o bin/govs.exe ./cmd
```

### 贡献代码

1. Fork 仓库
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 📌 注意事项

1. 🔐 环境变量修改需要管理员权限
2. 🔄 版本切换后需要重启终端/IDE
3. 💾 建议定期备份环境变量
4. ⚠️ 保留本地安装包在 down/ 目录
5. 📦 不要手动修改数据目录

## 🤝 参与贡献

- 提交前先检查现有问题
- 遵循代码风格
- 包含测试代码
- 更新相关文档
- 提供详细的 PR 描述

## 📄 开源协议

本项目采用 [MIT](./LICENSE) 开源协议。 