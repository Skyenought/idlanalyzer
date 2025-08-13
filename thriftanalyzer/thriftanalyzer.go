package thriftanalyzer

import (
	"fmt"
	"path/filepath"

	"github.com/joyme123/thrift-ls/lsp/lsputils"
	"github.com/joyme123/thrift-ls/parser"
	"go.lsp.dev/uri"
)

// AnalyzeThriftDependencies 分析 Thrift 文件的依赖关系，并返回一个包含丰富信息的图。
func AnalyzeThriftDependencies(mainIdlPath string, files map[string][]byte, options ...Option) (*RichDependencyGraph, error) {
	opts := newDefaultOptions()
	// Apply all provided functional options.
	for _, option := range options {
		option(opts)
	}
	pegParser := &parser.PEGParser{}
	asts := make(map[uri.URI]*parser.Document)

	cleanedFiles := make(map[string][]byte, len(files))
	for path, content := range files {
		cleanedFiles[filepath.Clean(path)] = content
	}
	mainIdlPath = filepath.Clean(mainIdlPath)

	graph := &RichDependencyGraph{
		Nodes:          make(map[string]*FileNode),
		EntryPointPath: mainIdlPath,
	}

	for path, content := range cleanedFiles {
		fileURI := uri.File(path)
		node := &FileNode{
			AbsolutePath: path,
			BaseName:     filepath.Base(path),
			IsEntryPoint: path == mainIdlPath,
			Includes:     []*DependencyEdge{},
			IncludedBy:   []*DependencyEdge{},
		}

		doc, parseErrs := pegParser.Parse(path, content)
		if len(parseErrs) > 0 {
			node.HasParseErrors = true
		}
		if doc != nil {
			asts[fileURI] = doc
			// 提取 Namespace 信息
			for _, ns := range doc.Namespaces {
				if ns.Language != nil && ns.Name != nil && ns.Language.Name != nil && ns.Name.Name != nil {
					node.Namespaces = extractNamespacesWithOptions(doc, opts)
				}
			}
		}
		graph.Nodes[path] = node
	}

	adj := make(map[string][]string) // 用于循环检测的邻接表
	for sourceURI, doc := range asts {
		sourcePath := sourceURI.Filename()
		sourceNode := graph.Nodes[sourcePath]

		for _, include := range doc.Includes {
			if include.Path == nil || include.Path.Value == nil {
				continue
			}

			rawPath := include.Path.Value.Text
			targetURI := lsputils.IncludeURI(sourceURI, rawPath)
			targetPath := targetURI.Filename()

			edge := &DependencyEdge{
				SourcePath:     sourcePath,
				TargetPath:     targetPath,
				RawIncludePath: rawPath,
				IsBroken:       graph.Nodes[targetPath] == nil, // 如果目标节点不存在，则标记为损坏
			}

			sourceNode.Includes = append(sourceNode.Includes, edge)
			if targetNode, ok := graph.Nodes[targetPath]; ok {
				targetNode.IncludedBy = append(targetNode.IncludedBy, edge)
			}

			// 填充邻接表
			if !edge.IsBroken {
				adj[sourcePath] = append(adj[sourcePath], targetPath)
			}
		}
	}

	// --- 第三遍: 分析问题 ---
	result := &AnalysisResult{
		Conflicts: detectNamespaceConflicts(asts),
	}

	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)
	for path := range cleanedFiles {
		if !visited[path] {
			detectCycleInSdk(path, visited, recursionStack, adj, []string{}, &result.Cycles)
		}
	}

	var returnErr error
	if !result.IsEmpty() {
		returnErr = result
	}

	return graph, returnErr
}

// detectNamespaceConflicts 检测所有类型的命名空间冲突。
func detectNamespaceConflicts(asts map[uri.URI]*parser.Document) NamespaceConflict {
	// 结构: map[namespace_identifier] -> map[scope] -> file_path
	definitions := make(map[string]map[string]string)
	conflicts := make(NamespaceConflict)

	for fileURI, doc := range asts {
		filePath := fileURI.Filename()

		// 检查显式定义的 namespace
		for _, ns := range doc.Namespaces {
			if ns.Language == nil || ns.Name == nil || ns.Language.Name == nil || ns.Name.Name == nil {
				continue
			}
			scope := ns.Language.Name.Text
			identifier := ns.Name.Name.Text

			if _, ok := definitions[identifier]; !ok {
				definitions[identifier] = make(map[string]string)
			}

			if existingFile, ok := definitions[identifier][scope]; ok {
				// 发现冲突
				if existingFile != filePath {
					conflictKey := fmt.Sprintf("Namespace '%s' (scope: %s)", identifier, scope)
					if _, exists := conflicts[conflictKey]; !exists {
						conflicts[conflictKey] = []string{existingFile}
					}
					// 避免重复添加同一个文件
					isAlreadyListed := false
					for _, f := range conflicts[conflictKey] {
						if f == filePath {
							isAlreadyListed = true
							break
						}
					}
					if !isAlreadyListed {
						conflicts[conflictKey] = append(conflicts[conflictKey], filePath)
					}
				}
			} else {
				definitions[identifier][scope] = filePath
			}
		}

		// 检查隐式命名空间 (基于文件名)
		scope := "(default)" // 用一个特殊标识符代表隐式作用域
		identifier := lsputils.GetIncludeName(fileURI)
		if _, ok := definitions[identifier]; !ok {
			definitions[identifier] = make(map[string]string)
		}
		if existingFile, ok := definitions[identifier][scope]; ok {
			if existingFile != filePath {
				conflictKey := fmt.Sprintf("隐式 Namespace '%s'", identifier)
				if _, exists := conflicts[conflictKey]; !exists {
					conflicts[conflictKey] = []string{existingFile}
				}
				conflicts[conflictKey] = append(conflicts[conflictKey], filePath)
			}
		} else {
			definitions[identifier][scope] = filePath
		}
	}

	return conflicts
}

// detectCycleInSdk (保持不变)
func detectCycleInSdk(u string, visited, recursionStack map[string]bool, adj map[string][]string, path []string, cycles *[]CyclicDependency) {
	visited[u] = true
	recursionStack[u] = true
	path = append(path, u)

	for _, v := range adj[u] {
		// 如果 v 不在 adj 中，说明它是一个叶子节点或者文件不存在，直接跳过
		if _, ok := adj[v]; !ok && !visited[v] {
			visited[v] = true // 标记为已访问，避免重复处理
			continue
		}
		if !visited[v] {
			detectCycleInSdk(v, visited, recursionStack, adj, path, cycles)
		} else if recursionStack[v] {
			var cyclePath CyclicDependency
			var inCycle bool
			for _, node := range path {
				if node == v {
					inCycle = true
				}
				if inCycle {
					cyclePath = append(cyclePath, node)
				}
			}
			cyclePath = append(cyclePath, v)
			*cycles = append(*cycles, cyclePath)
		}
	}
	recursionStack[u] = false
}

func extractNamespacesWithOptions(doc *parser.Document, opts *analysisOptions) []*NamespaceInfo {
	var infos []*NamespaceInfo

	collectAll := len(opts.scopes) == 1 && opts.scopes[0] == "*"

	scopesToCollect := make(map[string]struct{})
	if !collectAll {
		for _, s := range opts.scopes {
			scopesToCollect[s] = struct{}{}
		}
	}

	for _, ns := range doc.Namespaces {
		if ns.Language == nil || ns.Name == nil || ns.Language.Name == nil || ns.Name.Name == nil {
			continue
		}
		scope := ns.Language.Name.Text

		if collectAll {
			infos = append(infos, &NamespaceInfo{
				Scope:      scope,
				Identifier: ns.Name.Name.Text,
			})
		} else {
			if _, ok := scopesToCollect[scope]; ok {
				infos = append(infos, &NamespaceInfo{
					Scope:      scope,
					Identifier: ns.Name.Name.Text,
				})
			}
		}
	}

	return infos
}
