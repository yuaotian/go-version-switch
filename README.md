# go-version-switch

<div align="center">

[![Release](https://img.shields.io/github/v/release/yuaotian/go-version-switch?style=flat-square&logo=github&color=blue)](https://github.com/yuaotian/go-version-switch/releases/latest)
[![Go Version](https://img.shields.io/badge/go-%3E%3D%201.16-blue)](https://img.shields.io/badge/go-%3E%3D%201.16-blue)
[![Release Build](https://github.com/{owner}/{repo}/actions/workflows/release.yml/badge.svg)](https://github.com/{owner}/{repo}/actions/workflows/release.yml)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

🔄 A simple  Go version management tool designed for Windows

[简体中文](./README_CN.md) | English

</div>

## ✨ Features

- 🔍 Real-time display of current Go version information
- 📋 Manage multiple installed Go versions
- ⬇️ Automatic download and installation of official releases
- 🔄 Quick switching between different Go versions
- ⚙️ Smart system environment variable management
- 💾 Support for environment configuration backup and restore
- 🔒 Secure environment variable rollback mechanism
- 🌐 Multi-architecture support (x86/x64/arm/arm64)

## 🚀 Quick Start

### 📥 Installation

#### Method 1: Direct Download

Download the latest version from the [Releases](https://github.com/yuaotian/go-version-switch/releases) page.

#### Method 2: Build from Source

```bash
# Clone repository
git clone https://github.com/yuaotian/go-version-switch.git
cd go-version-switch

# Build
go build -o go-version-switch.exe ./cmd

# Add executable to PATH
# Recommended to copy the compiled file to C:\Program Files\go-version-switch\
```

### 🎯 Basic Usage

```bash
# View help information
go-version-switch -h

# Check current version
go-version-switch -version

# List all installed versions
go-version-switch -list

# List all versions before update version list
go-version-switch -list -update

# Install specific version
go-version-switch -install 1.19.5 

# Install specific version and architecture
go-version-switch -install 1.19.5 -arch x64

# Switch to specified version
go-version-switch -use 1.19.5

# Rollback environment variable configuration
go-version-switch -rollback
```



## 📁 Project Structure

```
go-version-switch/
├── 📂 cmd/
│   └── main.go                 # Program entry
├── 📂 internal/
│   ├── config/                # Configuration management
│   │   └── config.go         # Configuration handling
│   └── version/              # Version management
│       ├── common.go        # Common functions
│       ├── download.go      # Download functionality
│       ├── env.go          # Environment variable handling
│       ├── goversion.go    # Version information
│       ├── install.go      # Installation logic
│       ├── list.go        # Version listing
│       ├── releases.go    # Release management
│       └── version.go     # Version control
├── 📂 bin/
│   └── data/              # Runtime data
│       └── config/        # Configuration files
├── 📄 go.mod              # Dependency management
├── 📄 go.sum              # Dependency verification
└── 📝 README.md           # Project documentation
```

## ⚙️ System Requirements

- Windows 10/11
- Go 1.16+ (only for compilation)
- Administrator privileges (for modifying environment variables)
- Stable network connection (for downloading new versions)

## 🔧 Troubleshooting

### Common Issues

1. **Insufficient Permissions**
   ```bash
   Error: Administrator privileges required
   Solution: Run command prompt as administrator
   ```

2. **Download Failure**
   ```bash
   Error: Download timeout
   Solution: Check network connection or use proxy
   ```

3. **Version Switch Failure**
   ```bash
   Error: Environment variable update failed
   Solution: Use -rollback command to restore previous configuration
   ```

## 👨‍💻 Developer Guide

### Building the Project

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build and test
go build -v -o bin/go-version-switch.exe ./cmd/main.go && ./bin/go-version-switch -install 1.23.4 -arch x86
```

### Contributing

1. Fork the project
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📌 Important Notes

1. 🔐 Administrator privileges required for modifying system environment variables
2. 🔄 Terminal or IDE restart required after version switch
3. 💾 Regular backup of environment variable configuration recommended
4. ⚠️ Ensure stable network connection
5. 📦 Do not manually modify the tool's data directory

## 🤝 Contribution Guidelines

- Search for existing issues before submitting a new one
- Provide detailed descriptions for Pull Requests
- Follow project code standards
- Ensure submitted code is tested

## 📄 License

This project is licensed under the [MIT](./LICENSE) License. 