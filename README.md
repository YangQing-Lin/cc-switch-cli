# cc-switch-cli

一个轻量级的命令行工具，用于管理多个Claude API配置并支持快速切换。

## 功能特性

- 🔄 **快速切换** - 通过单个命令在不同的Claude API配置之间切换
- 📝 **配置管理** - 添加、删除和查看多个API配置
- 🔐 **安全存储** - 显示时API令牌会被遮掩，配置文件受权限保护
- 🌍 **跨平台** - 支持Windows、macOS、Linux和其他操作系统
- 💡 **交互式输入** - 支持命令行参数和交互式提示
- 🎨 **用户友好** - 清晰的列表显示和直观的状态指示器

## 安装

### 从源码构建

需要Go 1.25.1或更高版本：

```bash
git clone https://github.com/YangQing-Lin/cc-switch-cli.git
cd cc-switch-cli
go build -o cc-switch
```

### 下载预编译二进制文件

从[发布页面](https://github.com/YangQing-Lin/cc-switch-cli/releases)下载适合您操作系统的预编译二进制文件。

## 使用方法

### 列出所有配置

```bash
cc-switch
```

示例输出：
```
Configuration List:
─────────
● official              Token: sk-ant-...***  URL: https://api.anthropic.com  Category: official
○ 88code                Token: 88_e7...***   URL: https://www.example.org/api  Category: custom
○ local-proxy           Token: sk-ww...***   URL: http://127.0.0.1:3456  Category: custom
```

- `●` 表示当前激活的配置
- `○` 表示非激活配置

### 切换配置

```bash
cc-switch <配置名称>
```

示例：
```bash
cc-switch 88code
```

输出：
```
✓ Switched to configuration: 88code
  Token: 88_e7...***
  URL: https://www.example.org/api
```

### 添加新配置

#### 方法1：交互模式

```bash
cc-switch config add my-config
```

程序将提示您输入：
- API令牌（隐藏输入）
- 基础URL
- 类别（可选）

#### 方法2：命令行参数

```bash
cc-switch config add my-config \
  --apikey "sk-ant-xxxxx" \
  --base-url "https://api.example.com" \
  --category "custom"
```

支持的类别类型：
- `official` - 官方API
- `cn_official` - 官方中国区
- `aggregator` - 聚合服务
- `third_party` - 第三方服务
- `custom` - 自定义（默认）

### 删除配置

```bash
cc-switch config delete <配置名称>
```

添加`--force`或`-f`标志跳过确认：

```bash
cc-switch config delete my-config --force
```

## 配置文件

配置文件位置：

- **Windows**: `%USERPROFILE%\.cc-switch\config.json`
- **macOS/Linux**: `~/.cc-switch/config.json`

配置文件格式：
```json
{
  "version": 2,
  "claude": {
    "providers": {
      "uuid-xxx": {
        "id": "uuid-xxx",
        "name": "config-name",
        "settingsConfig": {
          "env": {
            "ANTHROPIC_AUTH_TOKEN": "your-api-token",
            "ANTHROPIC_BASE_URL": "api-endpoint"
          }
        },
        "category": "custom",
        "createdAt": 1234567890
      }
    },
    "current": "active-config-id"
  }
}
```

## 与cc-switch GUI版本的兼容性

cc-switch-cli与[cc-switch](https://github.com/YangQing-Lin/cc-switch) GUI版本完全兼容：

- ✅ 共享相同的配置文件格式
- ✅ 支持相同的配置结构
- ✅ 可以互换使用
- ✅ 配置更改实时同步

您可以同时使用CLI和GUI版本，因为它们读取和写入相同的配置文件。

## 安全注意事项

1. **文件权限** - 配置文件默认为600权限（仅所有者可读/写）
2. **令牌遮掩** - 显示时API令牌会自动遮掩
3. **备份机制** - 每次保存前自动创建`.bak`备份文件
4. **输入保护** - 配置期间API令牌输入被隐藏

## 常见问题

### Q: 如何从旧版本配置迁移？

A: cc-switch-cli自动检测并将v1配置文件迁移到v2格式。

### Q: 配置文件损坏怎么办？

A: 您可以从自动生成的`config.json.bak`备份文件恢复。

### Q: 支持哪些Claude API提供商？

A: 支持所有与Anthropic API格式兼容的服务，包括：
- 官方Claude API
- 各种中继服务
- 本地代理服务

### Q: 如何验证配置是否有效？

A: 添加配置时会执行基本验证（名称、令牌、URL格式）。实际连接测试在使用时进行。

## 开发

### 项目结构

```
cc-switch-cli/
├── main.go                 # 入口点
├── cmd/                    # 命令行界面
│   ├── root.go            # 根命令
│   ├── config.go          # 配置子命令
│   ├── add.go             # 添加配置
│   └── delete.go          # 删除配置
├── internal/              # 内部实现
│   ├── config/           # 配置管理
│   └── utils/            # 实用函数
└── go.mod                # 依赖管理
```

### 构建项目

```bash
# 为当前平台构建
go build -o cc-switch

# 为Windows构建
GOOS=windows GOARCH=amd64 go build -o cc-switch.exe

# 为macOS构建
GOOS=darwin GOARCH=amd64 go build -o cc-switch-darwin

# 为Linux构建
GOOS=linux GOARCH=amd64 go build -o cc-switch-linux
```

### 运行测试

```bash
go test ./...
```

## 许可证

MIT许可证

## 贡献

欢迎提交问题和拉取请求！

## 相关项目

- [cc-switch](https://github.com/YangQing-Lin/cc-switch) - 带有图形界面的GUI版本，用于配置管理