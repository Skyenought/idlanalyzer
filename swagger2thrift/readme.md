# package `swagger2thrift`

## 概述

`swagger2thrift` 包提供了将 OpenAPI（v3 和 v2/Swagger）API 规范转换为 Apache Thrift 接口定义的功能。

此转换过程的核心是，它首先将 OpenAPI 规范解析到 **`idl_ast`** 结构中——这是一个**基于 `abcoder` 框架 `UniAST` 理念设计的抽象语法树**。随后，这些结构化的 `idl_ast` 实例可以被 `thriftwriter` 等工具处理，生成最终的 Thrift 文件。

## 主要特性

-   **多版本兼容**: 自动检测并处理 OpenAPI v3 和 Swagger v2 格式。
-   **智能结构转换**: 将 OpenAPI 的 `schemas` 转换为 Thrift 的 `structs`、`enums` 和 `typedefs`。
-   **HTTP 注解保留**: API 的 `paths` 和 `parameters` 被转换为 Thrift 的 `service` 函数，并附加上特殊的 HTTP 注解（如 `(api.get="/users/{id}")`, `(api.query="limit")`），从而保留了原始的 RESTful 路由信息。
-   **代码组织优化**: 能够根据 OpenAPI 定义的命名约定，将生成的 Thrift 代码智能地拆分到不同的文件中，并自动处理 `include` 关系，使代码结构更清晰。
-   **目录隔离**: 当处理多个 OpenAPI 文件时，每个文件的输出都会被放置在独立的子目录中，从根本上解决了命名冲突问题。

## 使用指南

核心功能通过 `ConvertSpecsToThrift` 函数暴露。此函数封装了“读取 OpenAPI -> 转换为 `idl_ast` -> 生成 Thrift 代码”的完整流程。

```go
// 函数签名
func ConvertSpecsToThrift(specs map[string][]byte, options ...Option) (map[string][]byte, error)
```
-   **`specs`**: 一个 map，键为 OpenAPI 文件名，值为文件内容。
-   **`options`**: 可选参数，用于自定义转换过程（如设置 Go 的 `namespace`）。
-   **返回**: 一个 map，键为生成的 Thrift 文件的相对路径（包含自动创建的子目录），值为文件内容。

### 示例代码
```go
package main

import (
    "log"
    "os"
    "github.com/Skyenought/idlanalyzer/swagger2thrift"
)

func main() {
    // 1. 准备输入文件
    specs := make(map[string][]byte)
    usersSpec, _ := os.ReadFile("path/to/users_api.json")
    specs["users_api.json"] = usersSpec

    // 2. 调用转换函数
    generatedFiles, err := swagger2thrift.ConvertSpecsToThrift(specs)
    if err != nil {
        log.Fatalf("转换失败: %v", err)
    }

    // 3. (可选) 将生成的文件写入磁盘
    for path, content := range generatedFiles {
        // ... 写入文件的逻辑 ...
        log.Printf("成功生成文件: %s", path)
    }
}
```