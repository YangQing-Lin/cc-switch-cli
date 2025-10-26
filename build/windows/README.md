# Windows 构建资源

此目录包含用于构建 Windows 版本的资源文件，主要用于降低 Windows Defender 误报率。

## 文件说明

### icon.ico
- **用途**：应用程序图标
- **来源**：从 UI 项目（cc-switch）复制
- **格式**：Multi-resolution ICO（包含 16x16, 32x32, 48x48, 64x64 等多种尺寸）
- **大小**：约 60KB

### resource.rc
- **用途**：Windows 资源定义文件
- **内容**：
  - 图标资源引用
  - 版本信息（FileVersion, ProductVersion）
  - 字符串信息（CompanyName, ProductName, Copyright, FileDescription）
- **格式**：标准 Windows RC 格式
- **编译目标**：`.syso` 文件（Go 自动识别）

### generate_syso.sh
- **用途**：自动生成 Windows 资源对象文件（.syso）
- **执行时机**：GoReleaser 构建前（在 `.goreleaser.yml` 的 `before.hooks` 中调用）
- **工作流程**：
  1. 检测 `windres` 编译器（mingw-w64 工具链）
  2. 如果不存在，则使用 `goversioninfo`（纯 Go 实现）
  3. 生成 `resource_windows_amd64.syso` 文件到项目根目录
  4. Go 编译器在构建 Windows 版本时自动包含此文件

## 构建说明

### 手动构建

```bash
# 1. 生成资源文件
bash build/windows/generate_syso.sh

# 2. 编译 Windows 版本
GOOS=windows GOARCH=amd64 go build -o ccs.exe

# 3. 验证（可选）
file ccs.exe  # 应显示 "PE32+ executable"
strings ccs.exe | grep "CC Switch CLI"  # 应显示产品名
```

### GitHub Actions 自动构建

GoReleaser 会自动调用 `generate_syso.sh`，无需手动干预。

## 原理说明

### 为什么需要这些文件？

Windows Defender 使用机器学习模型判断文件是否可疑，判断依据包括：

1. ❌ **缺少元数据**：裸 exe 无版权、版本信息
2. ❌ **无签名**：未经数字签名的可执行文件
3. ❌ **符号表被剥离**：使用 `-s -w` 编译的二进制（类似恶意软件的混淆手法）

通过嵌入资源文件，我们提供了：

1. ✅ **完整元数据**：公司名、产品名、版权、版本号
2. ✅ **应用图标**：正规软件的标志
3. ✅ **文件描述**：明确说明程序用途

### 效果

- 大幅降低 Windows Defender 误报率（预计 60-80% 改善）
- exe 右键属性显示专业的详细信息
- 提升用户信任度

## 进阶方案

### 方案 A：代码签名
- 购买 Windows 代码签名证书（$100-300/年）
- 使用 `signtool.exe` 签名 exe/msi
- 效果：几乎 100% 解决误报

### 方案 B：MSI 安装包
- 使用 WiX Toolset 打包成 .msi
- MSI 格式本身就是"受信任"的信号
- 参考 UI 项目的 `src-tauri/wix/per-user-main.wxs`

## 维护说明

### 更新图标
如果需要更换图标，替换 `icon.ico` 文件即可。建议格式：
- 多分辨率 ICO（16x16, 32x32, 48x48, 64x64, 128x128, 256x256）
- 32-bit 色深（带透明通道）

### 更新版本信息
修改 `resource.rc` 文件中的以下字段：
- `FILEVERSION` 和 `PRODUCTVERSION` 宏定义
- `FileVersion` 和 `ProductVersion` 字符串

**注意**：实际发布时，GoReleaser 会通过 ldflags 注入版本号到 Go 代码，无需手动修改 RC 文件的版本号。

### 更新公司/产品信息
修改 `resource.rc` 和 `generate_syso.sh` 中的 `versioninfo.json` 模板。

## 故障排查

### 问题：编译时未包含资源

**现象**：
```bash
strings ccs.exe | grep "CC Switch CLI"  # 无输出
```

**原因**：
- `resource_windows_amd64.syso` 文件不存在或未在项目根目录
- Go 编译器未检测到 .syso 文件

**解决**：
```bash
# 确保 .syso 文件在项目根目录
ls resource_windows_amd64.syso

# 重新生成
bash build/windows/generate_syso.sh
```

### 问题：generate_syso.sh 执行失败

**现象**：
```
windres: command not found
goversioninfo: command not found
```

**解决**：
脚本会自动安装 `goversioninfo`，但需要 Go 环境。确保：
```bash
go version  # 应显示 Go 1.21+
```

### 问题：图标未显示

**原因**：
- icon.ico 文件损坏或格式不兼容
- RC 文件中的路径错误

**解决**：
```bash
# 验证图标文件
file icon.ico  # 应显示 "MS Windows icon resource"

# 检查 RC 文件路径
grep "icon.ico" resource.rc  # 应显示 IDI_ICON1 ICON "icon.ico"
```

## 参考资料

- [Windows 资源脚本语法](https://learn.microsoft.com/en-us/windows/win32/menurc/resource-compiler)
- [goversioninfo 工具](https://github.com/josephspurrier/goversioninfo)
- [Go Windows GUI 打包指南](https://github.com/tc-hib/winres)

---

*文档版本：1.0*
*最后更新：2025-10-26*
