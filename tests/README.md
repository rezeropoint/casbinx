# CasbinX 集成测试

CasbinX 专注于集成测试，通过完整的功能流程测试确保权限管理库的正确性和安全性。

## 🔧 测试恢复TODO

### P0 - 核心基础 (必须首先完成)
- [ ] `helpers/db_helper.go` - 数据库操作和引导权限初始化
- [ ] `helpers/permission_helper.go` - 权限定义和测试数据
- [ ] `integration/engine_test.go` - 基础功能测试 (用户权限、角色、租户)

### P1 - 安全验证 (核心安全机制)
- [ ] `integration/security_test.go` - 安全特性测试 (系统角色保护、防提权)

### P2 - 完整覆盖 (全面测试)
- [ ] `helpers/test_report.go` - 测试报告工具
- [ ] `helpers/coverage_analyzer.go` - 覆盖率分析工具
- [ ] `integration/edge_cases_test.go` - 边界条件和并发测试
- [ ] `integration/comprehensive_test.go` - 综合接口测试

### P3 - 配置文件
- [ ] `fixtures/rbac_model.conf` - Casbin RBAC 模型配置
- [ ] `docker-compose.test.yml` - 测试环境配置
- [ ] `test.env` - 环境变量配置

**重建建议**：按P0→P1→P2→P3顺序进行，每个优先级完成后验证测试通过再进行下一优先级。

## 测试架构

```
tests/
├── helpers/                 # 测试辅助函数
│   ├── db_helper.go        # 数据库操作和引导权限初始化
│   └── permission_helper.go # 权限定义和测试数据
├── fixtures/               # 测试配置文件
│   └── rbac_model.conf     # Casbin RBAC 模型
├── integration/            # 集成测试
│   ├── engine_test.go      # 基础功能测试
│   ├── security_test.go    # 安全特性测试
│   ├── edge_cases_test.go  # 边界条件测试
│   └── comprehensive_test.go # 综合集成测试
├── docker-compose.test.yml # 测试环境（无持久化）
├── test.env               # 测试环境变量
└── README.md              # 本文档
```

## 快速开始

### 1. 启动测试环境
```bash
# 启动 PostgreSQL 和 Redis（无持久化，每次都是干净环境）
docker-compose -f tests/docker-compose.test.yml up -d

# 检查服务状态
docker-compose -f tests/docker-compose.test.yml ps
```

### 2. 运行测试
```bash
# 环境检查
go test -v ./tests/integration/... -run=TestEnvironmentHealth

# 运行集成测试
go test -v ./tests/integration/...

# 生成覆盖率报告
make test-coverage
```

### 3. 清理环境
```bash
# 关闭测试环境
docker-compose -f tests/docker-compose.test.yml down
```

## 测试覆盖详情

### P0 - 核心基础测试
**基础功能测试** (`engine_test.go`)：
- 用户权限管理：授予、检查、撤销权限
- 角色管理：创建、分配、删除角色
- 租户初始化：管理员角色分配和权限验证
- 租户隔离：跨租户权限隔离验证

### P1 - 安全验证测试
**安全特性测试** (`security_test.go`)：
- 系统角色保护：系统角色不可修改
- 全局角色保护：跨租户角色访问控制
- 权限提升防护：防止自我提权和间接提权

### P2 - 完整覆盖测试
**边界条件测试** (`edge_cases_test.go`)：
- 无效输入、并发操作、性能测试

**综合集成测试** (`comprehensive_test.go`)：
- 用户权限管理：`GetEffectivePermissionsSecure`、`ClearUserPermissions`等
- 角色管理：`GetUserRoles`、`SetRolePermissions`等
- 批量权限检查：`CheckMultiplePermissions`、`HasAnyPermission`等
- 资源访问检查：`CanAccessResource`、`GetUserTenants`等

## 测试环境配置

### Docker 环境
- PostgreSQL:15433 (避免端口冲突)
- Redis:16380 (避免端口冲突)
- 无持久化卷，每次测试都是干净环境

### 环境变量

测试配置在 `test.env` 文件中：

```env
TEST_DB_DSN=postgres://test_user:test_password@localhost:15433/casbinx_test?sslmode=disable
TEST_REDIS_ADDR=localhost:16380
TEST_TENANT_001=tenant_test_001
TEST_TENANT_002=tenant_test_002
```

## 测试辅助工具

### 数据库助手 (`helpers/db_helper.go`)

提供测试数据库操作和引导权限初始化：

```go
// 获取测试配置
config := helpers.GetTestConfig(t)

// 全面环境检查（推荐在测试前使用）
err := helpers.CheckTestEnvironment(t, config)

// 清理测试数据并初始化引导权限
helpers.CleanupDB(t, config.Dsn)
helpers.InitializeBootstrapUser(t, config.Dsn)

// 获取测试用户/租户/角色
users := helpers.GetTestUsers()
tenants := helpers.GetTestTenants()
roles := helpers.GetTestRoles()

// 获取环境健康状态
health := helpers.GetEnvironmentHealth(t, config)
```

**关键功能**：
- `InitializeBootstrapUser()`: 为 `admin_test_001` 创建超级管理员权限
- 解决了 CasbinX 引擎缺少第一个超级用户的引导问题

### 权限助手 (`helpers/permission_helper.go`)

提供常用权限定义：

```go
// 系统权限
systemPerms := helpers.GetSystemPermissions()

// 普通权限
normalPerms := helpers.GetNormalPermissions()

// 预定义权限
docRead := helpers.CommonPermissions.DocumentRead
userWrite := helpers.CommonPermissions.UserWrite
```

### 测试报告工具 (`helpers/test_report.go`)
提供测试结果收集和报告生成功能。

### 覆盖率分析工具 (`helpers/coverage_analyzer.go`)
提供覆盖率报告分析和HTML报告生成功能。

## 测试最佳实践

### 1. 引导权限初始化
每个测试前必须：清理数据库 → 初始化引导权限 → 刷新策略

### 2. 测试隔离原则
- 每次测试都使用干净环境（无持久化卷）
- 测试用例间无依赖关系
- 使用 `admin_test_001` 作为统一的超级管理员

### 3. 安全测试重点
- 验证权限提升防护
- 测试租户隔离边界
- 确保系统角色保护机制

### 4. 错误验证模式
验证期望错误：检查错误不为nil → 验证错误类型匹配

## 运行测试

```bash
# 基本操作
docker-compose -f tests/docker-compose.test.yml up -d  # 启动测试环境
go test -v ./tests/integration/...                     # 运行所有测试
make test-coverage                                      # 生成覆盖率报告

# 运行特定测试
go test -v ./tests/integration/... -run=TestEngine_BasicPermissionFlow    # 基础功能
go test -v ./tests/integration/... -run=TestEngine_Comprehensive         # 综合测试
go test -v ./tests/integration/... -run=TestEnvironmentHealth            # 环境检查
```

## 故障排除

### 常见问题
1. **测试环境连接失败** → 检查Docker状态：`docker-compose -f tests/docker-compose.test.yml ps`
2. **端口冲突** → 确认PostgreSQL:15433和Redis:16380端口可用
3. **引导权限失败** → 确保`InitializeBootstrapUser()`正确执行

### 调试命令
```bash
go test -v ./tests/integration/...                      # 详细输出
go test -race ./tests/integration/...                   # 检测竞态条件
go test -v -run TestName ./tests/integration/...        # 运行特定测试
```

## 核心接口覆盖

**用户权限**：`GrantPermission`、`RevokePermission`、`CheckPermission`、`GetEffectivePermissionsSecure`、`ClearUserPermissions`
**角色管理**：`CreateRole`、`AssignRole`、`RemoveRole`、`GetUserRoles`、`GetRolePermissions`、`SetRolePermissions`
**批量检查**：`CheckMultiplePermissions`、`HasAnyPermission`、`HasAllPermissions`
**资源访问**：`CanAccessResource`、`GetAvailableActions`、`GetUserTenants`

详细接口列表参考原始测试文件的注释。

---

🎯 **重建目标**：恢复完整的权限管理库测试覆盖，确保核心功能和安全特性正常工作。