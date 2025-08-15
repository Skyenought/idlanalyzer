## 快速开始

该库的核心功能通过 `ConvertSpecsToThrift` 函数暴露。您只需提供一个包含规范文件内容的 map，即可得到一个包含生成的 Thrift 文件内容的 map。

### 核心概念

-   **聚合服务文件**: 所有在 OpenAPI/Swagger 规范中定义的操作 (paths) 都会被聚合到**一个**服务 (Service) 中。这个服务所在的 Thrift 文件被固定命名为 `service.thrift`。
-   **自定义**: 您可以对生成过程进行自定义，例如指定 Go 的命名空间或自定义聚合服务的名称。

## `ConvertSpecsToThrift` 函数使用指南

### 函数签名
```go
func ConvertSpecsToThrift(specs map[string][]byte, options ...Option) (map[string][]byte, error)
```

-   **`specs`**: 一个 `map[string][]byte` 类型的参数。
    -   `key`: 原始规范文件的**文件名** (例如, `"api.json"` 或 `"swagger.yaml"`)。这个文件名会被用来自动生成 `go` 命名空间。
    -   `value`: 文件的完整内容，以 `[]byte` 的形式提供。
-   **`options`**: (可选) 一个或多个功能选项，用于自定义转换过程。例如，使用 `WithNamespace("my_service")` 来手动指定 Go 命名空间。
-   **返回值**:
    -   `map[string][]byte`: 一个包含所有生成文件的 map。
        -   `key`: 生成的 Thrift 文件名 (例如, `"service.thrift"`, `"definitions.thrift"`)。
        -   `value`: 该 Thrift 文件的 `[]byte` 内容。
    -   `error`: 如果在转换过程中出现任何错误，则返回一个非 `nil` 的 error。
    -   **注意**: `service.thrift` 是包含聚合服务定义的固定文件名。

### 使用示例

下面的例子展示了如何读取一个 OpenAPI 文件，将其转换为 Thrift，并自定义服务名称。

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
	// 1. 读取您的 Swagger 或 OpenAPI 文件
	specContent, err := os.ReadFile("path/to/your/docs_swagger_2.json")
	if err != nil {
		log.Fatalf("错误：无法读取规范文件: %v", err)
	}

	// 2. 准备输入参数 `specs` map
	// map 的 key 是原始文件名，value 是文件内容。
	specs := map[string][]byte{
		"docs_swagger_2.json": specContent,
	}

	// 3. 调用核心转换函数并传入自定义选项
	// 您可以链式调用多个选项。
	generatedFiles, err := swagger2thrift.ConvertSpecsToThrift(
		specs,
		swagger2thrift.WithNamespace("myapi"),      // 自定义 Go 命名空间
		swagger2thrift.WithServiceName("MyApiService"), // 自定义服务名称
	)
	if err != nil {
		log.Fatalf("错误：转换规范至 Thrift 失败: %v", err)
	}

	// 4. 将生成的 Thrift 文件保存到磁盘
	outputDir := "./generated_thrifts"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("错误：无法创建输出目录: %v", err)
	}

	fmt.Println("文件生成成功:")
	for filename, content := range generatedFiles {
		filePath := filepath.Join(outputDir, filename)
		err := os.WriteFile(filePath, content, 0644)
		if err != nil {
			fmt.Printf("  - 写入文件 %s 失败: %v\n", filePath, err)
			continue
		}
		fmt.Printf("  - %s\n", filePath)
	}
}
```

### 功能选项 (Options)

#### `WithNamespace(namespace string)`
默认情况下，生成的 Thrift 文件中的 Go 命名空间（`namespace go ...`）会根据输入的文件名自动生成。您可以使用此选项来覆盖默认行为，指定一个自定义的命名空间。

**示例:**
```go
// 所有生成的 thrift 文件都会包含 "namespace go my_custom_api"
generatedFiles, err := swagger2thrift.ConvertSpecsToThrift(specs, swagger2thrift.WithNamespace("my_custom_api"))
```

#### `WithServiceName(serviceName string)`
默认情况下，所有 HTTP 端点会被聚合到一个名为 `HTTPService` 的服务中。使用此选项可以指定一个自定义的服务名称。

**示例:**
```go
// 在 service.thrift 文件中，服务将被定义为 "service MyCustomService { ... }"
generatedFiles, err := swagger2thrift.ConvertSpecsToThrift(specs, swagger2thrift.WithServiceName("MyCustomService"))
```