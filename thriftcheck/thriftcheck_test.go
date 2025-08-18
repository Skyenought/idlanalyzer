package thriftcheck

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Check(t *testing.T) {
	maps, _ := loadThriftFilesFromDir("testdata")
	diagnosticsMap, err := ThriftSyntaxCheck(context.Background(), maps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "分析过程中出现严重错误: %v\n", err)
		os.Exit(1)
	}

	totalIssues := 0

	if len(diagnosticsMap) == 0 {
		fmt.Println("没有文件被分析。")
		return
	}

	for filename, diagnostics := range diagnosticsMap {
		if len(diagnostics) > 0 {
			// 如果文件的诊断列表不为空，说明发现了问题。
			fmt.Printf("\n[+] 在文件 %s 中发现的问题:\n", filename)
			for _, diag := range diagnostics {
				totalIssues++
				// +1 是为了将0-based的LSP坐标转换为人类可读的1-based坐标。
				fmt.Printf("  - 严重性: %s\n", diag.Severity)
				fmt.Printf("  - 位  置: 第 %d 行, 第 %d 个字符\n", diag.Range.Start.Line+1, diag.Range.Start.Character+1)
				fmt.Printf("  - 消  息:  %s\n", diag.Message)
				fmt.Println("    -----------------")
			}
		}
	}

	if totalIssues > 0 {
		fmt.Printf("分析完成。共发现 %d 个问题。\n", totalIssues)
	}
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
