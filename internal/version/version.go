package version

// Version 当前版本
const Version = "1.0.0"

// BuildDate 构建日期（由编译时注入）
var BuildDate = "unknown"

// GitCommit Git 提交哈希（由编译时注入）
var GitCommit = "unknown"

// GetVersion 获取版本信息
func GetVersion() string {
	return Version
}

// GetBuildDate 获取构建日期
func GetBuildDate() string {
	return BuildDate
}

// GetGitCommit 获取 Git 提交哈希
func GetGitCommit() string {
	return GitCommit
}
