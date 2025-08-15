## 快速开始

该库的核心功能通过 `ConvertSpecsToThrift` 函数暴露。您可以提供一个包含**一个或多个**规范文件内容的 map，函数会处理所有文件，并返回一个包含所有生成的 Thrift 文件内容的 map。

### 核心概念

-   **多文件处理**: 函数会遍历并处理输入 `specs` map 中的所有条目。
-   **目录隔离与动态文件名**: 每个 OpenAPI 规范生成的所有 Thrift 文件（包括主文件和根据定义拆分出的子文件）都会被放置在一个以其源文件名命名的**独立子目录**中。例如，`users_api.yaml` 的所有产出都会在 `users_api/` 目录下。这从根本上解决了多文件转换时的命名冲突问题。
-   **自定义**: 您可以对生成过程进行自定义，例如为所有文件指定一个统一的 Go 命名空间或自定义服务名称。

## `ConvertSpecsToThrift` 函数使用指南

### 函数签名
```go
func ConvertSpecsToThrift(specs map[string][]byte, options ...Option) (map[string][]byte, error)
```

-   **`specs`**: 一个 `map[string][]byte` 类型的参数。
    -   `key`: 原始规范文件的**文件名** (例如, `"users_api.json"` 或 `"payments_api.yaml"`)。这个文件名会被用来命名生成的**输出子目录**。
    -   `value`: 文件的完整内容，以 `[]byte` 的形式提供。
-   **`options`**: (可选) 一个或多个功能选项，用于自定义转换过程。
-   **返回值**:
    -   `map[string][]byte`: 一个包含所有生成文件的 map。
        -   `key`: 生成的 Thrift 文件的**相对路径** (例如, `"users_api/users_api.thrift"` 或 `"users_api/common.thrift"`)。键名现在包含了自动生成的目录结构。
        -   `value`: 该 Thrift 文件的 `[]byte` 内容。
    -   `error`: 如果在转换过程中出现任何错误，则返回一个非 `nil` 的 error。

### 使用示例

下面的例子展示了如何读取**多个** OpenAPI 文件，将它们全部转换为 Thrift，并将生成的文件（现在带有目录结构）写入磁盘。

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	// 替换为您的实际包路径
	"github.com/Skyenought/idlanalyzer/swagger2thrift"
)

func main() {
	// 1. 准备输入参数 `specs` map，可以包含多个文件。
	specs := make(map[string][]byte)

	// 读取第一个文件
	usersSpec, err := os.ReadFile("path/to/your/users_api.json")
	if err != nil {
		log.Fatalf("错误：无法读取 users_api.json: %v", err)
	}
	specs["users_api.json"] = usersSpec

	// 读取第二个文件
	paymentsSpec, err := os.ReadFile("path/to/your/payments_api.yaml")
	if err != nil {
		log.Fatalf("错误：无法读取 payments_api.yaml: %v", err)
	}
	specs["payments_api.yaml"] = paymentsSpec


	// 2. 调用核心转换函数
	// 选项会对所有文件的转换过程生效
	generatedFiles, err := swagger2thrift.ConvertSpecsToThrift(
		specs,
		swagger2thrift.WithNamespace("my_microservices"), // 为所有文件设置统一的Go命名空间
	)
	if err != nil {
		log.Fatalf("错误：转换规范至 Thrift 失败: %v", err)
	}

	// 3. 将生成的 Thrift 文件保存到磁盘
	outputDir := "./generated_thrifts"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("错误：无法创建输出目录: %v", err)
	}

	fmt.Println("文件生成成功:")
	for filePath, content := range generatedFiles {
		// filePath 已经是 "users_api/users_api.thrift" 这样的相对路径
		fullPath := filepath.Join(outputDir, filePath)

		// 确保文件的父目录存在
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			fmt.Printf("  - 创建目录 %s 失败: %v\n", filepath.Dir(fullPath), err)
			continue
		}
		
		err := os.WriteFile(fullPath, content, 0644)
		if err != nil {
			fmt.Printf("  - 写入文件 %s 失败: %v\n", fullPath, err)
			continue
		}
		fmt.Printf("  - %s\n", fullPath)
	}
	// 预期的输出文件现在会包含目录结构：
	// - ./generated_thrifts/users_api/users_api.thrift
	// - ./generated_thrifts/payments_api/payments_api.thrift
	// 假设 users_api.json 中定义了 "common.User"，则还会生成：
	// - ./generated_thrifts/users_api/common.thrift
}
```

### 功能选项 (Options)

#### `WithNamespace(namespace string)`
为所有生成的 Thrift 文件设置统一的 Go 命名空间 (`namespace go ...`)。如果未提供，命名空间会根据每个输入文件的文件名自动生成。

**示例:**
```go
// 所有生成的 thrift 文件都会包含 "namespace go my_custom_api"
generatedFiles, err := swagger2thrift.ConvertSpecsToThrift(specs, swagger2thrift.WithNamespace("my_custom_api"))
```

#### `WithServiceName(serviceName string)`
为所有生成的服务设置一个统一的名称。请注意，这会将所有输入文件中的服务都命名为同一个名字，这在多数情况下可能不是理想选择。此选项在处理单个规范文件时更为有用。

**示例:**
```go
// 在 users_api/users_api.thrift 和 payments_api/payments_api.thrift 文件中，
// 服务都将被定义为 "service MyCustomService { ... }"
generatedFiles, err := swagger2thrift.ConvertSpecsToThrift(specs, swagger2thrift.WithServiceName("MyCustomService"))
```