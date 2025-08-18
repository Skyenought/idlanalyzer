package thriftwriter

import (
	"fmt"
	"os"
	"path/filepath"
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
