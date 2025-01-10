# go-version-switch

<div align="center">

[![Release](https://img.shields.io/github/v/release/yuaotian/go-version-switch?style=flat-square&logo=github&color=blue)](https://github.com/yuaotian/go-version-switch/releases/latest)
[![Go Version](https://img.shields.io/badge/go-%3E%3D%201.16-blue)](https://img.shields.io/badge/go-%3E%3D%201.16-blue)
[![Release Build](https://github.com/yuaotian/go-version-switch/actions/workflows/release.yml/badge.svg)](https://github.com/yuaotian/go-version-switch/actions/workflows/release.yml)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

🔄 Powerful Go Version Management Tool, Designed for Windows

English | [简体中文](./README_CN.md)

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
- 📦 Local installation package detection and usage
- 🔧 Automatic directory integrity verification
- 🔄 Smart architecture switching and local package detection

## 🚀 Quick Start

### 📥 Installation Methods

#### Method 1: Direct Download

1. Download the latest version from [Releases](https://github.com/yuaotian/go-version-switch/releases)
2. Extract to specified directory (recommended: C:\Program Files\go-version-switch\)
3. Add to PATH environment variable:
   ```powershell
   # Add to system environment variables and restart terminal
   setx /M PATH "%PATH%;C:\Program Files\go-version-switch"
   # Install specific version
   govs.exe -install 1.23.4 -arch x64
   # Switch to installed version
   govs.exe -use 1.23.4
   # Switch architecture
   govs.exe -arch x64
   # Rollback environment variables
   govs.exe -rollback
   # List all available versions
   govs.exe -list
   # Force update version list
   govs.exe -list -update
   # View help information
   govs.exe -help
   ```

#### Method 2: Build from Source

```bash
# Clone repository
git clone https://github.com/yuaotian/go-version-switch.git
cd go-version-switch

# Build
go build -o bin/govs.exe ./cmd

# Test installation
./bin/go-version-switch -install 1.23.4 -arch x64

# One-click build and test
go build -v -o bin/govs.exe ./cmd/main.go && ./bin/govs.exe -install 1.23.4 -arch x64
```

### 🎯 Basic Usage

```bash
# View help information
go-version-switch -h

# List all available versions
go-version-switch -list

# Force update version list
go-version-switch -list -update

# Install specific version
go-version-switch -install 1.23.4 -arch x64

# Switch to installed version
go-version-switch -use 1.23.4

# Direct architecture switching
go-version-switch -arch x64
go-version-switch -arch x86

# Rollback environment variables
go-version-switch -rollback
```

### 🔧 Advanced Features

#### Architecture Management
```bash
# Supported architecture list
x86, 386, 32       (32-bit)
x64, amd64, x86-64 (64-bit)
arm                (ARM)
arm64              (ARM64)

# Architecture switching with local package detection
go-version-switch -arch x64  # Automatically find and use local installation package
```

#### Local Package Support
- Automatic detection of installation packages in down/ directory
- Priority use of local installation packages
- Package integrity verification before installation

#### Environment Variable Management
- Automatic backup before modification
- Secure rollback mechanism
- Smart PATH management
- GOROOT and GOARCH handling

## 📁 Project Structure

```
go-version-switch/
├── 📂 cmd/
│   └── main.go                 # Program entry
├── 📂 internal/
│   ├── config/                # Configuration management
│   │   └── config.go         # Configuration processing
│   └── version/              # Version management
│       ├── common.go        # Common functions
│       ├── download.go      # Download functionality
│       ├── env.go          # Environment variable handling
│       ├── install.go      # Installation logic
│       ├── list.go        # Version list
│       ├── releases.go    # Release management
│       └── version.go     # Version control
├── 📂 bin/
│   └── data/              # Runtime data
│       ├── go-version/   # Go installation directory
│       ├── down/         # Download cache
│       ├── backup_env/   # Environment variable backup
│       └── config/       # Configuration files
├── 📄 go.mod              # Dependency management
└── 📝 README.md           # Project documentation
```

## ⚙️ System Requirements

- Windows 10/11
- Go 1.16+ (only for compilation)
- Administrator privileges
- Network connection (for downloads)

## 🔧 Troubleshooting

### Common Issues

1. **Permission Errors**
   ```bash
   Error: Administrator privileges required
   Solution: Run as administrator
   ```

2. **Download Issues**
   ```bash
   Error: Download failed
   Solution:
   - Check network connection
   - Use local package from down/ directory
   - Check proxy settings
   ```

3. **Version Switch Failure**
   ```bash
   Error: Environment variable update failed
   Solution:
   1. Use -rollback to restore
   2. Check file permissions
   3. Close all Go processes
   ```

4. **Directory Integrity**
   ```bash
   Error: Incomplete Go installation
   Solution:
   - Tool will automatically attempt to repair using local package
   - Check installation packages in down/ directory
   ```

## 👨‍💻 Development Guide

### Building

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build with version information
go build -ldflags="-X 'main.Version=v1.0.0'" -o bin/govs.exe ./cmd
```

### Contributing

1. Fork repository
2. Create feature branch
3. Commit changes
4. Push to branch
5. Create Pull Request

## 📌 Notes

1. 🔐 Environment variable modification requires administrator privileges
2. 🔄 Terminal/IDE restart required after version switch
3. 💾 Regular environment variable backup recommended
4. ⚠️ Keep local installation packages in down/ directory
5. 📦 Don't manually modify data directory

### 🔄 Terminal Environment Variable Refresh Methods

In some editors (like VSCode, IntelliJ IDEA), the integrated terminal might not automatically update environment variables. You can use these methods to refresh manually:

#### PowerShell Terminal
```powershell
# Method 1: Refresh environment variables
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

# Method 2: Use refreshenv command (requires Chocolatey)
refreshenv
```

#### CMD Terminal
```cmd
# Method 1: Use refreshenv (if Chocolatey is installed)
refreshenv

# Method 2: Reload environment variables
set PATH=%PATH%
```

## 🤝 Contributing

- Check existing issues before submitting
- Follow code style
- Include test code
- Update relevant documentation
- Provide detailed PR description

## 📄 License

This project is licensed under the [MIT](./LICENSE) License.
