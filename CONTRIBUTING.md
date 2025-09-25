# 贡献指南

感谢您对 CasbinX 项目的关注！我们欢迎各种形式的贡献。

## 🚀 快速开始

### 1. Fork 项目
点击页面右上角的 "Fork" 按钮，将项目复制到您的 GitHub 账户。

### 2. 克隆项目
```bash
git clone https://github.com/yourusername/casbinx.git
cd casbinx
```

### 3. 设置开发环境
```bash
# 安装依赖
go mod download

# 启动测试环境
docker-compose -f tests/docker-compose.test.yml up -d

# 验证环境
go test -v ./tests/integration/... -run=TestEnvironmentHealth
```

## 🔧 开发流程

### 1. 创建功能分支
```bash
git checkout -b feature/your-feature-name
```

### 2. 进行开发
- 遵循项目的代码风格
- 添加必要的测试
- 更新相关文档

### 3. 运行测试
```bash
# 格式化代码
go fmt ./...

# 静态分析
go vet ./...

# 运行测试
go test -v ./tests/integration/...

# 检查覆盖率
make test-coverage
```

### 4. 提交更改
```bash
git add .
git commit -m "feat: 添加新功能描述"
git push origin feature/your-feature-name
```

### 5. 创建 Pull Request
- 确保 CI 检查通过
- 填写清晰的 PR 描述
- 响应代码审查意见

## 📝 代码规范

### Go 代码风格
- 使用 `gofmt` 格式化代码
- 遵循 [Effective Go](https://golang.org/doc/effective_go.html) 规范
- 使用有意义的变量和函数名
- 添加适当的注释

### 提交信息格式
遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
type(scope): description

[optional body]

[optional footer]
```

类型示例：
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `test`: 测试相关
- `refactor`: 重构代码
- `perf`: 性能优化

### 测试要求
- 新功能必须包含测试
- 确保测试覆盖率不降低
- 所有测试必须通过
- 优先编写集成测试

## 🛡️ 安全考虑

CasbinX 是一个安全相关的项目，在开发时请注意：

- **不要引入安全漏洞**：仔细审查权限检查逻辑
- **防范提权攻击**：确保权限分配的安全性
- **保护敏感信息**：不要在日志中泄露敏感数据
- **遵循最小权限原则**：只授予必要的权限

## 📋 Issue 指南

### 报告 Bug
使用 Bug 模板，包含：
- 详细的问题描述
- 复现步骤
- 预期行为 vs 实际行为
- 环境信息（Go 版本、OS 等）
- 相关日志或错误信息

### 功能请求
使用 Feature Request 模板，包含：
- 功能描述和使用场景
- 预期的 API 设计
- 可能的实现方案
- 对现有功能的影响

## 🔍 代码审查

我们重视代码审查，以确保：
- 代码质量和可维护性
- 安全性和性能
- 与项目整体架构的一致性

请耐心等待审查，并积极响应审查意见。

## 📚 文档贡献

文档同样重要！您可以：
- 改进现有文档的清晰度
- 添加使用示例
- 翻译文档到其他语言
- 修复文档中的错误

## 🎯 项目重点

当前项目的重点领域：
1. **测试覆盖**：补充和完善集成测试
2. **性能优化**：提升权限检查性能
3. **安全增强**：加强安全防护机制
4. **文档完善**：提供更好的使用文档

## ❓ 获得帮助

如果您有疑问：
- 查看项目文档：[CLAUDE.md](./CLAUDE.md)
- 浏览现有 Issues
- 在 Discussions 中提问
- 查看 [tests/README.md](./tests/README.md) 了解测试

## 🎉 致谢

感谢所有贡献者的努力！每一个贡献都让 CasbinX 变得更好。

---

再次感谢您的贡献！🙏