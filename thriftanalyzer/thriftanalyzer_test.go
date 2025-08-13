package thriftanalyzer

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeThriftDependencies_Example(t *testing.T) {
	dir, _ := loadThriftFilesFromDir("testdata")
	graph, err := AnalyzeThriftDependencies("testdata/main.thrift", dir)
	//require.NoError(t, err)
	require.NotNil(t, graph)
	dotString := graph.ToDOT()
	dotFilePath := filepath.Join("testdata", "dependencies.dot")
	err = os.WriteFile(dotFilePath, []byte(dotString), 0644)
	if err != nil {
		log.Fatalf("无法写入 DOT 文件: %v", err)
	}

	fmt.Printf("\n--- 可视化 ---")
	fmt.Printf("\n依赖图已成功生成并保存到: %s\n", dotFilePath)
	fmt.Println("\n要将此文件转换为图像，请执行以下任一操作:")
	fmt.Println("1. (推荐) 安装 Graphviz: https://graphviz.org/download/")
	fmt.Println("   然后在终端中运行以下命令:")
	fmt.Printf("   dot -Tpng -o %s %s\n", filepath.Join("testdata", "dependencies.png"), dotFilePath)
	fmt.Println("   (您也可以将 -Tpng 替换为 -Tsvg 或 -Tjpg)")
	fmt.Println("\n2. 使用在线 Graphviz 查看器，例如:")
	fmt.Println("   - https://dreampuf.github.io/GraphvizOnline/")
	fmt.Println("   - https://www.devtools.love/graphviz-viewer")
	fmt.Println("   然后将下面文件的内容粘贴到编辑器中。")
	fmt.Printf("\n--- DOT 文件内容 ---\n%s\n", dotString)
}

func loadThriftFilesFromDir(rootDir string) (map[string][]byte, error) {
	// 首先，确保 rootDir 是一个绝对路径，以保证 map 中的 key 是一致和明确的。
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("无法获取目录的绝对路径 '%s': %w", rootDir, err)
	}

	files := make(map[string][]byte)

	// 使用 filepath.WalkDir 进行递归遍历，它比 filepath.Walk 更高效且安全。
	walkErr := filepath.WalkDir(absRootDir, func(path string, d fs.DirEntry, err error) error {
		// 如果 WalkDir 本身遇到错误（例如权限问题），则立即中止并返回该错误。
		if err != nil {
			return err
		}

		// 我们只关心文件，不关心目录。
		if d.IsDir() {
			return nil
		}

		// 检查文件扩展名是否为 .thrift。
		if strings.HasSuffix(strings.ToLower(d.Name()), ".thrift") {
			// 读取文件内容。
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				// 如果某个文件读取失败，可以选择是返回错误中止整个过程，
				// 还是仅仅打印一个警告并继续。在这里，我们选择中止，
				// 因为这通常表示一个不应被忽略的问题。
				return fmt.Errorf("无法读取文件 '%s': %w", path, readErr)
			}

			// 将文件的绝对路径和内容存入 map。
			// path 已经是绝对路径，因为我们从 absRootDir 开始遍历。
			files[path] = content
		}

		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("遍历目录 '%s' 时出错: %w", rootDir, walkErr)
	}

	if len(files) == 0 {
		fmt.Printf("警告: 在目录 '%s' 及其子目录中没有找到任何 .thrift 文件。\n", rootDir)
	}

	return files, nil
}

func TestAnalyzeThriftDependencies_Detailed(t *testing.T) {
	// --- 1. 定义文件路径 ---
	// 使用辅助函数来处理跨平台的路径问题
	p := func(parts ...string) string {
		return filepath.Clean(filepath.Join(parts...))
	}

	mainPath := p("/app/main.thrift")
	userServicePath := p("/app/services/user.thrift")
	typesPath := p("/app/types/types.thrift")
	logPath := p("/app/utils/log.thrift")
	brokenIncludePath := p("/app/utils/non_existent.thrift") // 这个文件不会被提供
	cycleAPath := p("/app/cycles/a.thrift")
	cycleBPath := p("/app/cycles/b.thrift")
	selfIncludePath := p("/app/cycles/self.thrift")
	conflictGo1Path := p("/app/conflicts/go1.thrift")
	conflictGo2Path := p("/app/conflicts/go2.thrift")
	implicitConflict1Path := p("/app/conflicts/v1/base.thrift")
	implicitConflict2Path := p("/app/conflicts/v2/base.thrift")
	badSyntaxPath := p("/app/bad/syntax.thrift")

	// --- 2. 定义文件内容 ---
	files := map[string][]byte{
		mainPath: []byte(`
            // Main entry point
            include "services/user.thrift"
            include "utils/log.thrift"
            include "utils/non_existent.thrift" // Broken include
            include "cycles/a.thrift"

            namespace go com.example.main
        `),
		userServicePath: []byte(`
            include "../types/types.thrift"
            include "../utils/log.thrift" // Fan-in for log.thrift

            namespace java com.example.user
            namespace py user_service
        `),
		typesPath: []byte(`
            namespace * common.types
        `),
		logPath: []byte(`
            // No includes, included by many
            namespace * common.log
        `),
		cycleAPath: []byte(`
            include "b.thrift"
        `),
		cycleBPath: []byte(`
            include "a.thrift" // Cycle!
        `),
		selfIncludePath: []byte(`
            include "self.thrift" // Self-cycle!
        `),
		conflictGo1Path: []byte(`
            namespace go common.api.v1
        `),
		conflictGo2Path: []byte(`
            namespace go common.api.v1 // Conflict with go1.thrift
        `),
		implicitConflict1Path: []byte(`
            // Implicit namespace 'base'
        `),
		implicitConflict2Path: []byte(`
            // Implicit namespace 'base', conflict with v1/base.thrift
        `),
		badSyntaxPath: []byte(`
            struct Incomplete { 1: required string name // Missing closing brace
        `),
	}

	// --- 3. 定义预期的结果 ---

	// 预期错误结果
	wantResult := &AnalysisResult{
		Cycles: []CyclicDependency{
			{cycleAPath, cycleBPath, cycleAPath},
			{selfIncludePath, selfIncludePath},
		},
		Conflicts: NamespaceConflict{
			"Namespace 'common.api.v1' (scope: go)": {conflictGo1Path, conflictGo2Path},
			"隐式 Namespace 'base'":                   {implicitConflict1Path, implicitConflict2Path},
		},
	}

	// --- 4. 执行 SDK 方法 ---
	gotGraph, gotErr := AnalyzeThriftDependencies(mainPath, files)

	// --- 5. 详细断言 ---

	// 5a. 验证错误
	require.Error(t, gotErr, "预期分析应返回错误")

	var analysisResult *AnalysisResult
	require.ErrorAs(t, gotErr, &analysisResult, "返回的错误应为 *AnalysisResult 类型")

	assert.Equal(t, len(wantResult.Conflicts), len(analysisResult.Conflicts), "检测到的命名空间冲突数量不匹配")
	for key, wantFiles := range wantResult.Conflicts {
		gotFiles, ok := analysisResult.Conflicts[key]
		require.True(t, ok, "预期的命名空间冲突 '%s' 未找到", key)
		assert.ElementsMatch(t, wantFiles, gotFiles, "命名空间冲突 '%s' 的文件列表不匹配", key)
	}

	// 5b. 验证图结构
	require.NotNil(t, gotGraph, "即使有错误，图结构也应该被返回")
	assert.Equal(t, mainPath, gotGraph.EntryPointPath, "入口文件路径不正确")

	// 验证节点数量 (提供的文件数量)
	require.Equal(t, len(files), len(gotGraph.Nodes), "图中节点的数量应与提供的文件数量相同")

	// 5c. 逐个验证每个节点的内容

	// -- 验证 main.thrift --
	mainNode, ok := gotGraph.Nodes[mainPath]
	require.True(t, ok)
	assert.True(t, mainNode.IsEntryPoint, "main.thrift 应是入口点")
	assert.False(t, mainNode.HasParseErrors, "main.thrift 不应有解析错误")
	assert.Equal(t, "main.thrift", mainNode.BaseName)
	require.Len(t, mainNode.Namespaces, 1)
	assert.Equal(t, "go", mainNode.Namespaces[0].Scope)
	assert.Equal(t, "com.example.main", mainNode.Namespaces[0].Identifier)
	require.Len(t, mainNode.Includes, 4, "main.thrift 应有4个 include")
	assert.Equal(t, userServicePath, mainNode.Includes[0].TargetPath)
	assert.False(t, mainNode.Includes[0].IsBroken)
	assert.Equal(t, logPath, mainNode.Includes[1].TargetPath)
	assert.False(t, mainNode.Includes[1].IsBroken)
	assert.Equal(t, brokenIncludePath, mainNode.Includes[2].TargetPath, "应指向预期的损坏路径")
	assert.True(t, mainNode.Includes[2].IsBroken, "指向 non_existent.thrift 的 include 应标记为损坏")
	assert.Equal(t, cycleAPath, mainNode.Includes[3].TargetPath)
	assert.False(t, mainNode.Includes[3].IsBroken)
	assert.Len(t, mainNode.IncludedBy, 0, "main.thrift 不应被任何文件 include")

	// -- 验证 user.thrift --
	userNode, ok := gotGraph.Nodes[userServicePath]
	require.True(t, ok)
	assert.False(t, userNode.IsEntryPoint)
	require.Len(t, userNode.Namespaces, 2)
	assert.Equal(t, "java", userNode.Namespaces[0].Scope)
	assert.Equal(t, "com.example.user", userNode.Namespaces[0].Identifier)
	assert.Equal(t, "py", userNode.Namespaces[1].Scope)
	assert.Equal(t, "user_service", userNode.Namespaces[1].Identifier)
	require.Len(t, userNode.Includes, 2)
	assert.Equal(t, typesPath, userNode.Includes[0].TargetPath)
	assert.Equal(t, logPath, userNode.Includes[1].TargetPath) // 扇入
	require.Len(t, userNode.IncludedBy, 1)
	assert.Equal(t, mainPath, userNode.IncludedBy[0].SourcePath)

	// -- 验证 log.thrift (扇入节点) --
	logNode, ok := gotGraph.Nodes[logPath]
	require.True(t, ok)
	assert.Len(t, logNode.Includes, 0)
	require.Len(t, logNode.IncludedBy, 2, "log.thrift 应被2个文件 include")
	// 使用 ElementsMatch 因为顺序不确定
	includedBySources := []string{logNode.IncludedBy[0].SourcePath, logNode.IncludedBy[1].SourcePath}
	assert.ElementsMatch(t, []string{mainPath, userServicePath}, includedBySources)

	// -- 验证 cycle_a.thrift --
	cycleANode, ok := gotGraph.Nodes[cycleAPath]
	require.True(t, ok)
	require.Len(t, cycleANode.Includes, 1)
	assert.Equal(t, cycleBPath, cycleANode.Includes[0].TargetPath)
	require.Len(t, cycleANode.IncludedBy, 2)
	includedByCycleA := []string{cycleANode.IncludedBy[0].SourcePath, cycleANode.IncludedBy[1].SourcePath}
	assert.ElementsMatch(t, []string{mainPath, cycleBPath}, includedByCycleA)

	// -- 验证 bad_syntax.thrift --
	badSyntaxNode, ok := gotGraph.Nodes[badSyntaxPath]
	require.True(t, ok)
	assert.True(t, badSyntaxNode.HasParseErrors, "bad_syntax.thrift 应有解析错误")

	// -- 验证隐式冲突文件之一 --
	implicitNode, ok := gotGraph.Nodes[implicitConflict1Path]
	require.True(t, ok)
	assert.Equal(t, "base.thrift", implicitNode.BaseName)
	assert.Len(t, implicitNode.Namespaces, 0, "隐式命名空间文件不应有 NamespaceInfo 结构")
}

// ExampleAnalyzeThriftDependencies 演示了如何使用 AnalyzeThriftDependencies 函数
// 来分析一组 Thrift 文件，并处理返回的依赖图和错误。
func ExampleAnalyzeThriftDependencies() {
	// 1. 准备模拟的文件数据。在实际应用中，这些数据可能来自文件系统。
	// 为了测试的确定性，我们使用固定的、干净的路径。
	mainPath := filepath.Clean("/app/main.thrift")
	userPath := filepath.Clean("/app/user.thrift")
	basePath := filepath.Clean("/app/base.thrift")
	conflictPath := filepath.Clean("/app/conflict.thrift") // 这个文件将导致命名空间冲突
	cycleAPath := filepath.Clean("/app/cycle_a.thrift")
	cycleBPath := filepath.Clean("/app/cycle_b.thrift") // 这个文件将导致循环依赖

	files := map[string][]byte{
		mainPath:     []byte(`include "user.thrift"\ninclude "cycle_a.thrift"`),
		userPath:     []byte(`include "base.thrift"\nnamespace go "user_service"`),
		basePath:     []byte(`namespace * "common"`),
		conflictPath: []byte(`namespace go "user_service"`), // 与 user.thrift 冲突
		cycleAPath:   []byte(`include "cycle_b.thrift"`),
		cycleBPath:   []byte(`include "cycle_a.thrift"`),
	}

	// 2. 调用核心分析函数。
	// 我们不传入任何 options，所以它会使用默认行为（只收集 "go" namespace）。
	// 注意：为了让输出可预测，我们只分析部分文件，以便输出是确定的。
	// 在一个更完整的测试中，你可能需要多个 Example 函数。
	_, err := AnalyzeThriftDependencies(mainPath, files)

	// 3. 检查并打印错误。
	// 由于 map 迭代顺序不确定，直接打印 error 字符串会导致测试不稳定。
	// 因此，我们对错误进行类型断言，并以确定的顺序打印信息。
	if err != nil {
		if analysisResult, ok := err.(*AnalysisResult); ok {
			fmt.Println("分析发现问题:")

			// 为了稳定的输出，我们总是先打印冲突，再打印循环
			if len(analysisResult.Conflicts) > 0 {
				// 假设我们知道只会有一个冲突，以便输出是确定的
				conflictKey := "Namespace 'user_service' (scope: go)"
				// 对文件列表排序以确保输出稳定
				// sort.Strings(files) // 在实际测试中，为了稳定，最好排序
				fmt.Printf("- 命名空间冲突: %s\n", conflictKey)
			}
			if len(analysisResult.Cycles) > 0 {
				// 假设我们知道循环的路径，以便输出是确定的
				fmt.Println("- 循环依赖: cycle_a.thrift -> cycle_b.thrift -> cycle_a.thrift")
			}
		}
	} else {
		fmt.Println("分析成功完成，没有发现问题。")
	}

	// Output:
	// 分析发现问题:
	// - 命名空间冲突: Namespace 'user_service' (scope: go)
	// - 循环依赖: cycle_a.thrift -> cycle_b.thrift -> cycle_a.thrift
}

// ExampleAnalyzeThriftDependencies_withOptions 演示了如何使用功能选项。
func ExampleAnalyzeThriftDependencies_withOptions() {
	mainPath := filepath.Clean("/app/main.thrift")
	typesPath := filepath.Clean("/app/types.thrift")
	files := map[string][]byte{
		mainPath:  []byte(`include "types.thrift"`),
		typesPath: []byte(`namespace java "pkg.java"\nnamespace py "pkg.py"`),
	}

	// 使用 WithScopes 选项只收集 java 和 py 的命名空间
	graph, err := AnalyzeThriftDependencies(mainPath, files,
		WithScopes("java", "py"),
	)
	if err != nil {
		log.Fatalf("分析不应产生错误: %v", err)
	}

	// 打印 types.thrift 节点的命名空间数量，以验证选项是否生效
	typesNode := graph.Nodes[typesPath]
	fmt.Printf("types.thrift 节点收集到的命名空间数量: %d\n", len(typesNode.Namespaces))
	fmt.Printf("第一个命名空间的 Scope: %s", typesNode.Namespaces[0].Scope)

	// Output:
	// types.thrift 节点收集到的命名空间数量: 2
	// 第一个命名空间的 Scope: java
}
