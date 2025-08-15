package swagger2thrift

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Skyenought/idlanalyzer/thriftwriter"

	"github.com/stretchr/testify/require"
)

func TestConvertOpenAPIToIDLSchema(t *testing.T) {
	file, _ := os.ReadFile("docs_swagger_2.json")
	schema, err := convertInternal("docs_swagger_2.json", file, nil)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// 2. 调用 Generate 函数
	generatedFiles, err := thriftwriter.Generate(schema)
	if err != nil {
		panic(err)
	}

	writeFiles("./generated_thrifts", generatedFiles)
}

func ExampleConvertSpecsToThrift() {
	file, _ := os.ReadFile("docs_swagger.yaml")
	maps := make(map[string][]byte)
	maps["docs_swagger.yaml"] = file
	toThrift, err := ConvertSpecsToThrift(maps)
	if err != nil {
		panic(err)
	}
	fmt.Println(toThrift)
}

func Test_ConvertSpecsToThrift(t *testing.T) {
	file, _ := os.ReadFile("fg_v2_infrastructure_controller_docs_swagger.json")
	maps := make(map[string][]byte)
	maps["fg_v2_infrastructure_controller_docs_swagger.json"] = file
	toThrift, err := ConvertSpecsToThrift(maps)
	if err != nil {
		panic(err)
	}
	fmt.Println(toThrift)
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
