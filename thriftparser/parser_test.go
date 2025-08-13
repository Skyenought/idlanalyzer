// thriftparser/parser_test.go

package thriftparser

import (
	"encoding/json"
	"os"
	"path/filepath"
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
