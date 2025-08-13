package thriftanalyzer

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NamespaceInfo 存储了单个 namespace 声明的详细信息。
type NamespaceInfo struct {
	Scope      string `json:"scope"`      // 例如 "go", "java", "*"
	Identifier string `json:"identifier"` // 例如 "com.example.user"
}

// DependencyEdge 代表一个 'include' 关系（图中的一条边）。
type DependencyEdge struct {
	SourcePath     string `json:"sourcePath"`     // 包含 include 语句的文件的绝对路径
	TargetPath     string `json:"targetPath"`     // 被 include 的文件的绝对路径
	RawIncludePath string `json:"rawIncludePath"` // include 语句中的原始字符串
	IsBroken       bool   `json:"isBroken"`       // TargetPath 是否是一个未找到或无法解析的文件
}

// FileNode 代表一个 Thrift 文件（图中的一个节点）。
type FileNode struct {
	AbsolutePath   string            `json:"absolutePath"`
	BaseName       string            `json:"baseName"`
	Namespaces     []*NamespaceInfo  `json:"namespaces"`
	Includes       []*DependencyEdge `json:"includes"`   // 该文件包含的其他文件（出度）
	IncludedBy     []*DependencyEdge `json:"includedBy"` // 包含该文件的其他文件（入度）
	HasParseErrors bool              `json:"hasParseErrors"`
	IsEntryPoint   bool              `json:"isEntryPoint"`
}

// RichDependencyGraph 是更详细的依赖图结构。
type RichDependencyGraph struct {
	Nodes          map[string]*FileNode `json:"nodes"`          // Key: 文件的绝对路径
	EntryPointPath string               `json:"entryPointPath"` // 分析的入口文件
}

type CyclicDependency []string
type NamespaceConflict map[string][]string

type AnalysisResult struct {
	Cycles    []CyclicDependency
	Conflicts NamespaceConflict
}

// Error 方法使 AnalysisResult 满足 error 接口。
func (r *AnalysisResult) Error() string {
	if r.IsEmpty() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Thrift 依赖分析发现问题:\n")

	if len(r.Cycles) > 0 {
		sb.WriteString(fmt.Sprintf(" - 发现 %d 个循环依赖:\n", len(r.Cycles)))
		for _, cycle := range r.Cycles {
			var pathParts []string
			for _, p := range cycle {
				pathParts = append(pathParts, filepath.Base(p))
			}
			sb.WriteString(fmt.Sprintf("   - 循环路径: %s\n", strings.Join(pathParts, " -> ")))
		}
	}

	if len(r.Conflicts) > 0 {
		sb.WriteString(fmt.Sprintf(" - 发现 %d 个命名空间冲突:\n", len(r.Conflicts)))
		for conflictKey, files := range r.Conflicts {
			var baseFiles []string
			for _, f := range files {
				baseFiles = append(baseFiles, filepath.Base(f))
			}
			sb.WriteString(fmt.Sprintf("   - %s 被以下文件共用: %s\n", conflictKey, strings.Join(baseFiles, ", ")))
		}
	}

	return sb.String()
}

// IsEmpty 检查结果中是否包含任何问题。
func (r *AnalysisResult) IsEmpty() bool {
	return len(r.Cycles) == 0 && len(r.Conflicts) == 0
}
