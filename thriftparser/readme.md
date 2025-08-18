# package `thriftparser`

## 概述

`thriftparser` 包是 IDL 分析器工具套件的数据输入端。它的核心职责是读取和解析 Apache Thrift 源代码，并将其转换为一个结构化的内存表示——**抽象语法树（AST）**。

这个 AST 严格遵循 `idl_ast` 包中的定义，而 **`idl_ast` 本身是 `abcoder` 框架中 `UniAST` 设计思想的一个具体实现**。因此，`thriftparser` 是将 Thrift 语言接入 `abcoder` 式代码分析生态的第一步。

## 主要特性

-   **高保真解析**: 底层利用 `joyme123/thrift-ls` 的解析引擎，确保了对 Thrift 语法的全面且精准的支持。
-   **依赖关系解析**: 能够正确处理 `include` 指令，构建出一个覆盖整个项目的、关联完整的 AST，准确解析跨文件的类型引用。
-   **元数据保留**: 可以配置保留源代码中的重要信息：
    -   **注释**: 将文档和行内注释与它们所描述的 AST 节点关联起来。
    -   **代码位置**: 记录每个语法元素在源文件中的精确位置（行、列、偏移），为高级分析和代码重写工具提供基础。
-   **灵活的数据源**:
    -   `NewParser(rootDir)`: 从文件系统目录中自动发现并解析所有 `.thrift` 文件。
    -   `NewParserFromMap(fileMap)`: 从内存中的文件 map 进行解析，非常适合在无文件系统的环境（如测试或在线服务）中使用。
-   **标准输出**: 解析的最终产出是一个 `*idl_ast.IDLSchema` 对象，这是整个工具套件使用的标准数据格式。

## 使用指南

典型的使用流程是：创建一个 `ThriftParser` 实例，调用其 `ParseIDLs()` 方法，然后对返回的 `IDLSchema` 对象进行后续处理。

### 示例代码
```go
package main

import (
	"fmt"
	"github.com/Skyenought/idlanalyzer/thriftparser"
)

func main() {
	// 1. 创建一个解析器实例，指向包含 .thrift 文件的目录
	parser, err := thriftparser.NewParser("path/to/your/thrifts")
	if err != nil {
		panic(fmt.Sprintf("创建解析器失败: %v", err))
	}

	// 2. 执行解析过程
	schema, err := parser.ParseIDLs()
	if err != nil {
		panic(fmt.Sprintf("解析 IDL 失败: %v", err))
	}

	// 3. 使用生成的 AST
	fmt.Printf("解析完成！共处理 %d 个文件。\n", len(schema.Files))

	// 示例：打印第一个文件中的服务数量
	if len(schema.Files) > 0 {
		firstFile := schema.Files
		fmt.Printf("文件 '%s' 中包含 %d 个服务定义。\n", firstFile.Path, len(firstFile.Definitions.Services))
	}
}
```