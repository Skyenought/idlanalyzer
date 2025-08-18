# package `thriftwriter`

## 概述

`thriftwriter` 包扮演着与 `thriftparser` 相反的角色。它接收一个内存中的抽象语法树（AST）——即 `*idl_ast.IDLSchema` 实例——并将其序列化为格式规范的 Apache Thrift 源代码。

由于 `idl_ast` **是基于 `abcoder` 框架 `UniAST` 理念设计的**，`thriftwriter` 因此成为了将 `abcoder` 式的、结构化的代码表示重新物化为 `.thrift` 文件的关键组件。

## 主要特性

-   **从 AST 生成代码**: 能够精确地将 `IDLSchema` 的结构和内容翻译成符合 Thrift 语法的文本。
-   **格式化输出**: 生成的代码会自动缩进和格式化，确保了高度的可读性和风格一致性。
-   **完整语法支持**: 支持所有 Thrift 定义的生成，包括 `namespace`, `include`, `service`, `struct`, `union`, `exception`, `enum`, `const` 和 `typedef`。
-   **注释保留**: 默认情况下，存储在 `idl_ast` 中的注释会被一并写入输出文件，从而完整地保留了代码文档。此功能可通过选项禁用。
-   **多文件处理**: 如果输入的 `IDLSchema` 包含了多个 `File` 结构，`Generate` 函数将一次性返回一个包含所有对应文件内容的 map。

## 使用指南

核心功能通过 `Generate` 函数提供。

```go
// 函数签名
func Generate(schema *idl_ast.IDLSchema, opts ...Option) (map[string][]byte, error)```

-   **`schema`**: 指向要被写入的 `*idl_ast.IDLSchema` 对象的指针。
-   **`opts`**: 可选参数，用于自定义生成行为，例如 `WithNoComments(true)`。
-   **返回**: 一个 map，键为 `.thrift` 文件的相对路径，值为其生成的字节内容。

### 示例代码：解析、修改再写回

```go
package main

import (
    "fmt"
    "github.com/Skyenought/idlanalyzer/thriftparser"
    "github.com/Skyenought/idlanalyzer/thriftwriter"
)

func main() {
    // 1. 解析项目，得到 abcoder 风格的 AST
    parser, _ := thriftparser.NewParser("./thrifts")
    schema, _ := parser.ParseIDLs()

    // 2. 在内存中修改 AST
    // 假设我们要给 Person 结构体添加一个注释
    personStructs := schema.FindStructsByFQN("Person")
    if len(personStructs) > 0 {
        comment := idl_ast.Comment{Text: "/** This is a user profile structure. */"}
        personStructs.Comments = append([]idl_ast.Comment{comment}, personStructs.Comments...)
    }

    // 3. 使用 thriftwriter 将修改后的 AST 写回
    generatedFiles, err := thriftwriter.Generate(schema)
    if err != nil {
        panic(fmt.Sprintf("生成 Thrift 代码失败: %v", err))
    }

    // 4. (可选) 查看或保存生成的文件
    for path, content := range generatedFiles {
        fmt.Printf("--- 生成的文件: %s ---\n", path)
        fmt.Println(string(content))
    }
}
```