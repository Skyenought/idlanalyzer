package swagger2thrift

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestConvertOpenAPIToIDLSchema(t *testing.T) {
	dir, err := loadFilesFromDir("./testdata")
	assert.Nil(t, err)
	thrift, err := ConvertSpecsToThrift(dir)
	assert.Nil(t, err)
	writeFiles("testdata/thrifts", thrift)
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

func loadFilesFromDir(rootDir string) (map[string][]byte, error) {
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("无法获取目录的绝对路径 '%s': %w", rootDir, err)
	}

	files := make(map[string][]byte)

	walkErr := filepath.WalkDir(absRootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			// 如果某个文件读取失败，可以选择是返回错误中止整个过程，
			// 还是仅仅打印一个警告并继续。在这里，我们选择中止，
			// 因为这通常表示一个不应被忽略的问题。
			return fmt.Errorf("无法读取文件 '%s': %w", path, readErr)
		}

		files[path] = content

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
