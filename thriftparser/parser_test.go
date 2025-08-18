// thriftparser/parser_test.go

package thriftparser

import (
	"encoding/json"
	"fmt"
	"github.com/Skyenought/idlanalyzer/thriftwriter"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThriftParser_ParseIDLs(t *testing.T) {
	abs, err := filepath.Abs("testdata/thrifts")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	parser, err := NewParser(abs, WithNoComments(false), WithNoLocation(false))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Parse testdata thrifts",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser
			schema, err := p.ParseIDLs()
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIDLs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && schema == nil {
				t.Error("ParseIDLs() returned nil schema without error")
				return
			}

			// 简单的断言，验证解析是否基本正确
			if len(schema.Files) != 4 {
				t.Errorf("Expected 3 files to be parsed, but got %d", len(schema.Files))
			}

			// 打印 JSON 结果以供调试
			jsonData, _ := json.MarshalIndent(schema, "", "  ")
			os.WriteFile("tmp.json", jsonData, 0o644)
		})
	}
}

func ExampleNewParserFromMap() {
	fromDir, _ := loadThriftFilesFromDir("../thriftparser/testdata/thrifts")
	parser, err := NewParserFromMap("", fromDir, WithSortDefinitions(true))
	if err != nil {
		panic(err)
	}
	schema, err := parser.ParseIDLs()
	if err != nil {
		panic(err)
	}

	generatedFiles, err := thriftwriter.Generate(schema)
	if err != nil {
		panic(err)
	}

	writeFiles("./generated_thrifts", generatedFiles)
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
