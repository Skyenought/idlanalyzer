## 概述

该库的核心特性是它能够分析一整套相互依赖的文件，正确解析 `include` 指令并验证跨文件的类型引用。

## 核心功能

-   **全面的语法解析**: 使用 PEG (Parsing Expression Grammar) 解析器，确保文件遵循 Thrift 的官方语法。
-   **完整的语义分析**: 不仅仅检查语法，还会检查逻辑和语义层面的错误，包括：
    -   **未定义的类型引用**: 检测一个结构体、枚举或类型别名在没有被定义或包含的情况下就被使用。
    -   **名称冲突**: 在同一作用域内查找重复的结构体、服务、枚举等定义。
    -   **类型不匹配**: 验证字段的默认值类型是否与其定义的类型相符。
-   **循环依赖检测**: 识别并报告文件之间循环的 `include` 引用（例如，`a.thrift` 包含 `b.thrift`，而 `b.thrift` 又包含 `a.thrift`）。
-   **字段 ID 验证**: 检查结构体、联合体和异常中的字段 ID 是否重复或无效（例如，非正数）。
-   **内存分析**: 接收一个从文件名到其字节内容的 `map` 作为输入，在分析过程中无需访问文件系统。这使其具有高度的可移植性和效率。
-   **结构化的、机器可读的输出**: 返回一个详细的诊断信息 `map`，使得以编程方式处理分析结果变得非常容易。


### 示例代码

这是一个完整的示例，演示了如何从一个目录加载所有 Thrift 文件并进行分析。

```go
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"your/import/path/thriftcheck" // <-- 导入你的包
)

// 辅助函数：从指定目录递归加载所有 .thrift 文件。
func loadThriftFilesFromDir(rootDir string) (map[string][]byte, error) {
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("无法获取目录 '%s' 的绝对路径: %w", rootDir, err)
	}

	files := make(map[string][]byte)
	walkErr := filepath.WalkDir(absRootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".thrift") {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return fmt.Errorf("无法读取文件 '%s': %w", path, readErr)
			}
			files[path] = content
		}
		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("遍历目录 '%s' 时出错: %w", rootDir, walkErr)
	}
	return files, nil
}

func main() {
	// 1. 将你所有的 thrift 文件加载到一个 map 中。
	// Key 必须是文件的完整路径，这样 `include` 指令才能被正确解析。
	sources, err := loadThriftFilesFromDir("testdata")
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载 thrift 文件时出错: %v\n", err)
		os.Exit(1)
	}

	// 2. 使用 sources map 调用检查器。
	diagnosticsMap, err := thriftcheck.ThriftSyntaxCheck(context.Background(), sources)
	if err != nil {
		// 这个错误表示分析过程本身失败了。
		fmt.Fprintf(os.Stderr, "分析过程中出现严重错误: %v\n", err)
		os.Exit(1)
	}

	// 3. 处理结构化的结果。
	totalIssues := 0
	fmt.Println("--- Thrift 分析报告 ---")

	for filename, diagnostics := range diagnosticsMap {
		if len(diagnostics) > 0 {
			totalIssues += len(diagnostics)
			fmt.Printf("\n[+] 在文件 %s 中发现的问题:\n", filename)
			for _, diag := range diagnostics {
				// +1 是为了将从0开始的LSP坐标转换为人类可读的从1开始的行号/字符位置。
				fmt.Printf("    - 严重性: %s\n", diag.Severity)
				fmt.Printf("    - 位  置: 第 %d 行, 第 %d 个字符\n", diag.Range.Start.Line+1, diag.Range.Start.Character+1)
				fmt.Printf("    - 消  息:  %s\n", diag.Message)
				fmt.Println("    -----------------")
			}
		} else {
			fmt.Printf("\n[✔] 文件 %s 中未发现问题。\n", filename)
		}
	}

	fmt.Println("\n--- 报告结束 ---")
	if totalIssues > 0 {
		fmt.Printf("分析完成。共发现 %d 个问题。\n", totalIssues)
		// os.Exit(1) // 可以选择在CI环境中用错误码退出
	} else {
		fmt.Println("分析完成。所有文件均有效。")
	}
}
```

## API 参考

### `ThriftSyntaxCheck`

```go
func ThriftSyntaxCheck(ctx context.Context, sources map[string][]byte) (map[string][]protocol.Diagnostic, error)
```

-   **`ctx context.Context`**: 用于控制取消操作的上下文。
-   **`sources map[string][]byte`**: 输入的从文件名到文件内容的 `map`。为确保 `include` 能被可靠地解析，文件名应该是绝对路径。
-   **返回 `map[string][]protocol.Diagnostic`**: 一个 `map`，其中每个键都是输入中的文件名，值是在该文件中找到的所有诊断信息的切片。如果一个文件没有问题，其对应的切片将为空。
-   **返回 `error`**: 一个非 `nil` 的错误表示分析设置过程中出现了严重失败（例如，某个检查器内部出现bug），而不是源文件中的验证错误。

### `protocol.Diagnostic`

这个结构体提供了关于每个问题的详细信息。关键字段包括：

-   **`Range protocol.Range`**: 问题在文件中的确切位置，包含 `Start` 和 `End` 两个点（行和字符，从0开始计数）。
-   **`Severity protocol.DiagnosticSeverity`**: 问题的严重性（错误、警告、信息或提示）。
-   **`Message string`**: 对错误的人类可读的描述。
-   **`Source string`**: 指示问题的来源（例如 "thrift-ls"）。
