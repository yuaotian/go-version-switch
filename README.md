# go-version-switch

<div align="center">

![Version](https://img.shields.io/badge/version-1.0.0-blue)
![Go Version](https://img.shields.io/badge/go-%3E%3D%201.16-blue)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

🔄 一个简单而强大的 Go 版本管理工具，专为 Windows 系统打造

[English](./README_EN.md) | 简体中文

</div>

## ✨ 特性

- 🔍 实时显示当前 Go 版本信息
- 📋 管理多个已安装的 Go 版本
- ⬇️ 自动下载安装官方发布版本
- 🔄 快速切换不同 Go 版本
- ⚙️ 智能管理系统环境变量
- 💾 支持环境配置备份恢复

## 🚀 快速开始

### 📥 安装

```bash
# 克隆仓库
git clone https://github.com/yuaotian/go-version-switch.git
cd go-version-switch

# 编译
go build -o go-version-switch.exe ./cmd

# 添加到 PATH 环境变量
```

### 🎯 使用示例

```bash
# 查看当前版本
go-version-switch -version
# 输出示例：
# Current Go Version: go1.16.5

# 查看可用版本
go-version-switch -list
# 输出示例：
# Installed Go Versions:
# ✓ 1.16.5 (current)
#   1.17.3
#   1.18.1

# 安装新版本
go-version-switch -install 1.19.5
# 输出示例：
# Downloading Go 1.19.5...
# Installation complete!

# 切换版本
go-version-switch -use 1.19.5
# 输出示例：
# Switching to Go 1.19.5...
# Successfully switched!

# 备份环境配置
go-version-switch -backup
# 输出示例：
# Environment variables backed up to: ./backup_20230615.json

# 恢复环境配置
go-version-switch -restore ./backup_20230615.json
# 输出示例：
# Environment variables restored successfully!
```

## 📁 项目结构

```
go-version-switch/
├── 📂 cmd/
│   └── main.go              # 程序入口
├── 📂 internal/
│   ├── config/             # 配置管理
│   ├── version/            # 版本控制
│   └── ...
├── 📄 go.mod               # 依赖管理
└── 📝 README.md            # 项目文档
```

## ⚙️ 配置要求

- Windows 10/11
- Go 1.16+
- 管理员权限（用于修改环境变量）

## 📌 注意事项

1. 🔐 需要管理员权限来修改系统环境变量
2. 🔄 切换版本后请重启终端或 IDE
3. 💾 建议定期备份环境变量配置
4. ⚠️ 确保网络连接稳定以下载新版本

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

## 📄 开源协议

本项目采用 [MIT](./LICENSE) 开源协议。 