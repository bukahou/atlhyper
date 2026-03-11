// atlhyper_master_v2/github/types.go
// GitHub 集成模块 — 类型定义
package github

import "time"

// ConnectionStatus GitHub 连接状态
type ConnectionStatus struct {
	Connected      bool   `json:"connected"`
	AccountLogin   string `json:"accountLogin"`
	AvatarURL      string `json:"avatarUrl"`
	InstallationID int64  `json:"installationId"`
}

// Repository 仓库信息
type Repository struct {
	FullName      string `json:"fullName"`
	DefaultBranch string `json:"defaultBranch"`
	Private       bool   `json:"private"`
}

// AuthorizedRepo 已授权仓库（含映射状态）
type AuthorizedRepo struct {
	Repository
	MappingEnabled bool `json:"mappingEnabled"`
}

// Commit 提交信息
type Commit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	AvatarURL string    `json:"avatarUrl"`
	Date      time.Time `json:"date"`
}

// CompareResult commit 比较结果
type CompareResult struct {
	HTMLURL string        `json:"htmlUrl"` // GitHub compare 页面 URL
	Files   []ChangedFile `json:"files"`
}

// PullRequest PR 信息
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

// ChangedFile 变更文件
type ChangedFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"` // added, modified, removed
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// FileEntry 目录条目
type FileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // file, dir
	Path string `json:"path"`
}

// CodeSearchResult 代码搜索结果
type CodeSearchResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Config GitHub 客户端配置
type Config struct {
	AppID          int64
	AppSlug        string // GitHub App URL slug (e.g. "atlhyper")
	PrivateKeyPath string
	CallbackURL    string
}

// Installation GitHub App 安装记录
type Installation struct {
	InstallationID int64  `json:"installationId"`
	AccountLogin   string `json:"accountLogin"`
}
