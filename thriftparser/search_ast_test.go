package thriftparser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Skyenought/idlanalyzer/idl_ast"
	"github.com/Skyenought/idlanalyzer/thriftwriter"
)

func TestIDLSchema_FindByFQN(t *testing.T) {
	parser, err := NewParser("../thriftparser/testdata/thrifts")
	if err != nil {
		panic(err)
	}
	schema, err := parser.ParseIDLs()
	if err != nil {
		panic(err)
	}
	person := schema.FindStructsByFQN("Person")
	person[0].Comments = append(person[0].Comments, idl_ast.Comment{
		Text: "// 你在干什么",
	})
	greeter := schema.FindServicesByFQN("Greeter")
	greeter[0].Name = "TestGreeter"

	generatedFiles, err := thriftwriter.Generate(schema, thriftwriter.WithNoComments(false))
	if err != nil {
		panic(fmt.Sprintf("生成代码失败: %v", err))
	}
	fmt.Printf("✅ 成功生成了 %d 个文件的内容。\n", len(generatedFiles))

	outputDir := "./generated_thrifts"
	writeFiles(outputDir, generatedFiles)
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
