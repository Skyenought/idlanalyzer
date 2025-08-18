package thriftwriter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Skyenought/idlanalyzer/thriftparser"
)

func TestWriter(t *testing.T) {
	parser, err := thriftparser.NewParser("../thriftparser/testdata/thrifts", thriftparser.WithSortDefinitions(true))
	if err != nil {
		panic(err)
	}
	schema, err := parser.ParseIDLs()
	if err != nil {
		panic(err)
	}

	// 2. 调用 Generate 函数
	generatedFiles, err := Generate(schema, WithNoComments(false))
	if err != nil {
		panic(err)
	}

	writeFiles("./generated_thrifts", generatedFiles)
}

func Test_NewParserFromMap(t *testing.T) {
	fromDir, _ := loadThriftFilesFromDir("../thriftparser/testdata/thrifts")
	parser, err := thriftparser.NewParserFromMap("", fromDir, thriftparser.WithSortDefinitions(true))
	if err != nil {
		panic(err)
	}
	schema, err := parser.ParseIDLs()
	if err != nil {
		panic(err)
	}

	// 2. 调用 Generate 函数
	generatedFiles, err := Generate(schema, WithNoComments(false))
	if err != nil {
		panic(err)
	}

	writeFiles("./generated_thrifts", generatedFiles)
}

func writeFiles(outputDir string, generatedFiles map[string][]byte) {
	for relativePath, content := range generatedFiles {
		destPath := filepath.Join(outputDir, relativePath)

		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			fmt.Printf("❌ 错误：无法创建目录 %s: %v\n", destDir, err)
			continue
		}

		err := os.WriteFile(destPath, content, 0o644)
		if err != nil {
			fmt.Printf("❌ 错误：无法写入文件 %s: %v\n", destPath, err)
			continue
		}

		fmt.Printf("✅ 成功将修改后的内容写入到: %s\n", destPath)
	}
}

func loadThriftFilesFromDir(rootDir string) (map[string][]byte, error) {
	files := make(map[string][]byte)

	walkErr := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(strings.ToLower(d.Name()), ".thrift") {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				// 如果某个文件读取失败，可以选择是返回错误中止整个过程，
				// 还是仅仅打印一个警告并继续。在这里，我们选择中止，
				// 因为这通常表示一个不应被忽略的问题。
				return fmt.Errorf("无法读取文件 '%s': %w", path, readErr)
			}

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
