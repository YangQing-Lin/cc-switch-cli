# Antivirus Optimization Summary

## 已实施措施
- 生成 Windows 资源 (`build/windows/generate_syso.sh`)：使用 `goversioninfo`/`windres` 写入版本号、发布者、图标与 manifest，减少“未知发布者”类启发式拦截。
- 提供 Inno Setup 安装程序：`setup.iss` 构建的用户级安装包（开始菜单快捷方式、可选桌面图标、最低权限），默认输出 `ccs-<version>-Setup.exe`。
- Release 流水线更新：Windows 构建与安装包在同一 Release 上传，保留独立的 `ccs-windows-amd64.exe` 以方便无安装场景。

## 效果与预期
- 更完整的元数据与用户级安装路径，有助于降低杀毒与 SmartScreen 的误报率。
- 安装器模式将图标、版本和发布者信息集中呈现，减少裸 exe 被误报的概率，同时避免需要管理员权限。

## 参考
- docs/go-antivirus-optimization-guide.md
