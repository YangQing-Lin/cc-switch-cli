//go:build no_self_update

package version

import "fmt"

// SelfUpdateEnabled 指示当前构建是否启用自更新能力
func SelfUpdateEnabled() bool { return false }

// CheckForUpdate 在禁用自更新的构建中不可用
func CheckForUpdate() (*ReleaseInfo, bool, error) {
	return nil, false, fmt.Errorf("自更新在本构建中已禁用")
}

// DownloadUpdate 在禁用自更新的构建中不可用
func DownloadUpdate(_ *ReleaseInfo) error {
	return fmt.Errorf("自更新在本构建中已禁用")
}

// InstallBinary 在禁用自更新的构建中不可用
func InstallBinary(_ string, _ bool) error {
	return fmt.Errorf("自更新在本构建中已禁用")
}

// GetReleasePageURL 返回 Release 页面，供提示用户手动下载
func GetReleasePageURL() string { return githubReleaseURL }
