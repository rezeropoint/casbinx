# CasbinX

CasbinX 是一个基于 Casbin 的安全权限管理引擎，实现了分级权限控制和全面的安全保护机制。项目专注于防范提权漏洞，提供系统级权限保护，支持多租户环境下的精细化权限管理。

## 🚀 快速开始

### 1. 启动测试环境

```bash
# 启动 PostgreSQL 和 Redis
docker-compose -f tests/docker-compose.test.yml up -d
```

### 2. 验证项目配置

```bash
# Windows PowerShell
.\test.ps1

# 或手动验证
go build ./...
go test -v ./tests/integration/... -run=TestNone
```

### 3. 运行集成测试

```bash
# 运行所有集成测试
go test -v ./tests/integration/...

# 或使用 Makefile (Linux/macOS)
make test
```

### 4. 导入使用

```go
package main

import (
    "casbinx/core"
    "casbinx/engine"
)

func main() {
    config := core.Config{
        Dsn: "postgres://user:pass@localhost/db?sslmode=disable",
        PossiblePaths: []string{"./rbac_model.conf"},
        Security: core.DefaultSecurityConfig(),
    }

    casbinx, err := engine.NewCasbinx(config)
    // 使用 CasbinX 进行权限管理...
}
```

## 🏗️ 核心特性

### 安全保护机制
- ✅ **防止自我提权**：阻止用户给自己分配管理权限
- ✅ **系统权限保护**：核心系统权限完全不可变更
- ✅ **租户隔离**：严格的多租户数据隔离
- ✅ **角色保护**：系统角色不可修改，只能在初始化时分配
- ✅ **操作者验证**：所有权限操作都验证操作者身份

### 权限管理功能
- 🔐 **分级权限**：普通权限 vs 系统权限
- 👥 **角色继承**：支持复杂的角色权限继承
- 🏢 **多租户**：完整的租户隔离和管理
- 📊 **权限查询**：安全的权限查询接口
- 🔄 **实时同步**：基于 Redis 的多实例权限同步

## 📋 基本使用

### 创建 CasbinX 引擎

```go
package main

import (
    "casbinx/core"
    "casbinx/engine"
)

func main() {
    config := core.Config{
        Dsn: "postgres://user:pass@localhost/db?sslmode=disable",
        PossiblePaths: []string{"./rbac_model.conf"},
        Security: core.DefaultSecurityConfig(),
        Watcher: core.WatcherConfig{
            Redis: core.RedisWatcherConfig{
                Addr:    "localhost:6379",
                Channel: "casbinx_sync",
            },
        },
    }

    casbinx, err := engine.NewCasbinx(config)
    if err != nil {
        panic(err)
    }

    // 使用 CasbinX...
}
```

### 权限管理示例

```go
// 授予权限
err := casbinx.GrantPermission(
    "admin_001",           // 操作者
    "user_001",            // 目标用户
    "company_001",         // 租户
    core.Permission{
        Resource: "document",
        Action:   core.ActionRead,
    },
)

// 检查权限
hasPermission, err := casbinx.CheckPermission(
    "user_001",
    "company_001",
    core.Permission{Resource: "document", Action: core.ActionRead},
)

// 角色管理
err = casbinx.AssignRole("admin_001", "user_001", "editor", "company_001")
```

## 🧪 测试

项目专注于集成测试，确保完整业务流程的正确性：

```bash
# 启动测试环境
docker-compose -f tests/docker-compose.test.yml up -d

# 运行集成测试
go test -v ./tests/integration/...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./tests/integration/...
go tool cover -html=coverage.out -o coverage.html
```

### 使用 Makefile

```bash
make test             # 运行测试
make test-coverage    # 生成覆盖率报告
```

## 📁 项目结构

```
casbinx/
├── core/                    # 核心类型和安全验证器
│   ├── config.go           # 配置定义
│   ├── security.go         # 安全验证器
│   ├── permission.go       # 权限相关类型
│   └── ...
├── engine/                  # CasbinX 主要接口
│   ├── engine.go           # 接口定义
│   └── handler.go          # 实现逻辑
├── internal/                # 内部实现模块
│   ├── check/              # 权限检查
│   ├── policy/             # 策略管理
│   ├── role/               # 角色管理
│   └── user/               # 用户权限管理
└── tests/                   # 集成测试
    ├── helpers/            # 测试辅助函数
    ├── fixtures/           # 测试配置文件
    └── integration/        # 集成测试用例
```

## 🔧 开发

### 环境要求
- Go 1.25+
- PostgreSQL 15+ (用于测试)
- Redis 7+ (用于测试)
- Docker & Docker Compose (用于测试)

### 开发命令

```bash
# 构建项目
go build ./...

# 代码格式化
go fmt ./...

# 静态分析
go vet ./...

# 依赖管理
go mod tidy
```

### 使用 Makefile

```bash
make help           # 显示所有命令
make build          # 构建项目
make fmt            # 格式化代码
make vet            # 静态分析
make dev-test       # 开发时快速测试
make ci             # 完整 CI 流程
```

## 📖 文档

- [CLAUDE.md](./CLAUDE.md) - 项目架构和开发指南
- [tests/README.md](./tests/README.md) - 集成测试文档

## 🔒 安全特性

### 权限分级
- **普通权限**：可自由授予和撤销的日常操作权限
- **系统权限**：不可变更的核心系统权限（租户管理、系统配置等）

### 安全机制
- **防自我提权**：阻止用户给自己分配管理权限
- **系统角色保护**：包含系统权限的角色完全不可修改
- **租户隔离**：严格的多租户数据隔离
- **操作者验证**：所有权限管理操作都验证操作者权限

## 🤝 贡献

我们欢迎各种形式的贡献！请查看 [贡献指南](CONTRIBUTING.md) 了解详情。

## 📄 许可证

本项目采用 Apache 2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。