# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

CasbinX 是一个基于 Casbin 的安全权限管理引擎，实现了分级权限控制和全面的安全保护机制。项目专注于防范提权漏洞，提供系统级权限保护，支持多租户环境下的精细化权限管理。

## 项目架构

### 三层架构设计

#### Engine 层 (`engine/`)
- **CasbinX接口**：统一的权限管理 API 入口
- **核心方法**：用户权限管理、角色分配、权限检查、租户初始化
- **安全集成**：所有权限操作都需要 `operatorKey` 参数进行操作者身份验证

#### Internal 层 (`internal/`)
- **用户管理器** (`user/`)：处理用户权限相关操作
- **角色管理器** (`role/`)：处理角色和角色权限操作，集成系统角色保护
- **检查管理器** (`check/`)：处理权限验证和查询
- **策略管理器** (`policy/`)：处理 Casbin 策略的底层操作

#### Core 层 (`core/`)
- **Enforcer**：封装 Casbin 执行器，提供基础权限操作
- **SecurityValidator**：安全验证器，实施安全策略和权限类型判断
- **配置和类型定义**：Permission、Role、Config 等核心数据结构

## 权限系统设计

### 权限类型分级
1. **普通权限** (`PermissionTypeNormal`)：可自由授予和撤销的日常操作权限
2. **系统权限** (`PermissionTypeSystem`)：完全不可变更的核心系统权限，包括：
   - 租户管理权限 (`tenant`)
   - 系统配置权限 (`system`)
   - 用户管理权限 (`user`)
   - 权限管理权限 (`permission`)

### 角色类型识别
- **系统角色**：包含系统权限的角色（如 `super_admin`, `admin`）
- **普通角色**：不包含系统权限的角色（如 `user`, `tenant_manager`）

## 核心安全特性

1. **系统角色完全保护**：包含系统权限的角色完全不可修改
2. **系统角色分配保护**：系统角色只能在租户初始化时分配
3. **全局域角色保护**：保护在全局域（*）中分配的角色
4. **租户初始化安全验证**：验证管理员角色的安全性
5. **防止自我提权**：防止用户给自己分配管理权限
6. **操作者权限验证**：所有权限管理操作都验证操作者权限

## 重要类型定义

### Permission 结构体
```go
type Permission struct {
    Resource Resource `json:"resource"` // 资源类型
    Action   Action   `json:"action"`   // 操作类型
}
```

### 基础操作类型
- `ActionRead`：读取/查看权限
- `ActionWrite`：创建/更新/配置权限
- `ActionDelete`：删除权限

### 系统资源常量
- `ResourceTenant`：租户资源
- `ResourceSystem`：系统资源
- `ResourceUser`：用户资源
- `ResourcePermission`：权限资源
- `ResourceRole`：角色资源

## API 使用模式

所有权限管理操作都需要提供 `operatorKey` 参数：

```go
// 用户权限管理
casbinx.GrantPermission(operatorKey, userKey, tenantKey, permission)
casbinx.RevokePermission(operatorKey, userKey, tenantKey, permission)

// 角色分配
casbinx.AssignRole(operatorKey, userKey, roleKey, tenantKey)
casbinx.RemoveRole(operatorKey, userKey, roleKey, tenantKey)

// 租户初始化（专门的安全方法）
casbinx.InitializeTenant(tenantKey, adminUserKey, adminRoleKey)

// 安全权限查询
casbinx.GetDirectPermissionsSecure(operatorKey, userKey, tenantKey)
casbinx.GetEffectivePermissionsSecure(operatorKey, userKey, tenantKey)
```

## 错误处理

项目使用自定义错误类型，提供友好的错误信息：

- `ErrSystemRoleImmutable`：系统角色不可修改
- `ErrSystemRoleAssignmentDenied`：系统角色分配被拒绝
- `ErrSelfElevationPrevented`：防止自我提权
- `ErrGlobalRoleAccessDenied`：全局角色访问被拒绝

## 安全配置

推荐使用 `core.DefaultSecurityConfig()` 获取默认安全配置：

```go
config := core.Config{
    Dsn: "postgres://user:pass@localhost/db?sslmode=disable",
    Security: core.DefaultSecurityConfig(),
}
```

## 数据库驱动注意事项

在使用临时结构体时，使用 `sql.NullString` 或 `sql.NullTime` 防止数据库驱动在 Scan 时报错：`converting NULL to string is unsupported`。这是因为多数 Postgres 驱动（如 lib/pq）不会把 NULL 安全地写成 nil 到目标 `*string`。

## 开发命令

项目使用标准 Go 工具链进行开发：

```bash
# 构建项目
go build ./...

# 运行集成测试
go test -v ./tests/integration/...

# 模块管理
go mod tidy
go mod download

# 代码格式化
go fmt ./...

# 静态分析
go vet ./...
```

### 使用 Makefile（推荐）

项目提供了完整的 Makefile 来简化开发流程：

```bash
# 启动测试环境
make test-env-up

# 运行集成测试
make test

# 生成测试覆盖率报告
make test-coverage

# 关闭测试环境
make test-env-down

# 开发工作流
make dev-test          # 格式化 + 静态检查 + 测试
make ci                # 完整 CI 流程
```

### 测试策略

项目专注于集成测试，确保完整业务流程的正确性：
- **集成测试**：位于 `tests/integration/`，测试完整的权限管理流程
- **测试环境**：使用 Docker Compose 管理 PostgreSQL 和 Redis
- **测试辅助**：`tests/helpers/` 提供数据库清理和权限定义辅助函数

## 最佳实践

1. **使用默认安全配置**：除非有特殊需求，建议使用 `core.DefaultSecurityConfig()`
2. **明确区分操作者和目标用户**：在所有权限操作中，明确指定操作者（operatorKey）
3. **创建合适的租户管理员角色**：不包含租户权限和系统权限，但有足够的权限管理租户内资源
4. **最小权限原则**：只授予必要的权限，避免过度授权
5. **使用安全查询接口**：对敏感权限查询使用 `*Secure` 版本的接口

## 快速开始

### 1. 启动测试环境
```bash
# 使用 Docker Compose 启动依赖服务
docker-compose -f tests/docker-compose.test.yml up -d

# 或使用开发环境
docker-compose -f docker-compose.dev.yml up -d
```

### 2. 运行测试验证
```bash
# 快速验证项目配置（Windows）
.\test.ps1

# 或手动运行集成测试
go test -v ./tests/integration/...
```

### 3. 查看示例代码
```bash
# 基础使用示例
cd examples/basic_usage && go run main.go

# 多租户示例
cd examples/multi_tenant && go run main.go
```

## 项目文件结构

```
casbinx/
├── core/                    # 核心类型和安全验证器
├── engine/                  # CasbinX 主要接口实现
├── internal/                # 内部实现模块
│   ├── check/              # 权限检查管理器
│   ├── policy/             # Casbin 策略管理
│   ├── role/               # 角色管理器
│   └── user/               # 用户权限管理器
├── tests/                  # 测试相关
│   ├── helpers/            # 测试辅助函数
│   ├── fixtures/           # 测试数据和配置
│   └── integration/        # 集成测试
├── examples/               # 使用示例
└── scripts/                # 数据库初始化脚本
```