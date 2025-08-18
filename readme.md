# IDL Analyzer Toolkit

它**基于 `abcoder` 框架的核心设计原则**开发，专注于接口定义语言（IDL）文件的深度分析、转换与生成，尤其擅长处理 Apache Thrift 和 OpenAPI 规范。

该工具套件的核心是一个通用的抽象语法树（AST）表示——**`idl_ast`**。这个结构是 `abcoder` 框架中 **`UniAST`** 概念的一个专门化实现，经过精心设计，旨在为 Thrift 和 OpenAPI 提供一个结构化、类型安全且易于编程式操作的中间表示。

## 主要特性

-   **OpenAPI 到 Thrift 转换**: 将 OpenAPI v2 (Swagger) 和 v3 规范无缝转换为结构严谨的 Thrift 文件，便于将 RESTful API 集成到 RPC 生态中。
-   **Thrift 解析与生成**: 包含一个的解析器，可将 Thrift 源码转换为 `idl_ast` 结构；以及一个写入器，可从 `idl_ast` 实例反向生成格式优美的 Thrift 代码。
-   **受 `abcoder` 启发的 AST**: 使用 `idl_ast` 这一通用 AST 结构，将所有工具的逻辑与源 IDL 格式解耦，这遵循了 `abcoder` 的核心设计哲学，极大地增强了工具的可扩展性。
-   **深度依赖分析**: 能够构建 Thrift 项目的完整依赖图，精准检测循环依赖和命名空间（`namespace`）冲突。
-   **编程式修改能力**: 通过操作内存中的 `idl_ast`，开发者可以实现复杂的自动化任务，如代码重构、批量添加注解或动态生成数据结构。

## 工具套件组件

本仓库遵循模块化设计，各个组件职责清晰，协同工作。

| 包 | 描述 |
| --- | --- |
| **[`idl_ast/`](#idl_ast)** | 定义了 `idl_ast` 结构，这是 **`abcoder` `UniAST` 概念的一个具体实现**，也是整个工具套件的基石。 |
| **[`thriftparser/`](#thriftparser)** | 提供了将 Thrift 源文件解析为 `idl_ast` 实例的功能。 |
| **[`thriftwriter/`](#thriftwriter)** | 负责将 `idl_ast` 实例写回为格式化的 `.thrift` 源代码文件。 |
| **[`thriftanalyzer/`](#thriftanalyzer)** | 提供了对 Thrift 项目进行静态分析的工具，如依赖图构建和冲突检测。 |
| **[`swagger2thrift/`](#swagger2thrift)** | 包含了将 OpenAPI (v2/v3) 规范转换为 `idl_ast` 表示的完整逻辑。 |

---
### <a name="idl_ast"></a> `idl_ast/`

此包是整个工具套件的核心。它定义了一个通用的 AST 结构，旨在以一种与语言无关的方式表示任何 IDL。所有其他工具都基于此结构运行。

-   **功能**:
    -   为 IDL 的所有元素（如 `Service`, `Message` (struct, union, exception), `Enum`, `Function`, `Field` 等）定义了 Go 结构体。
    -   提供了搜索和查询功能来导航 AST，例如 `FindServicesByFQN` 或 `FindStructsByFQN`，允许按完全限定名称查找定义。
-   **详细文档**: AST 的完整 JSON 结构规范位于 [`decs.md`](idl_ast/decs.md) 文件中。

---
### <a name="thriftparser"></a> `thriftparser/`

解析器负责读取 Thrift 源代码并将其转换为 `idl_ast` 中定义的 AST。

-   **功能**:
    -   基于 `joyme123/thrift-ls` 解析器，以确保高兼容性。
    -   处理 `include` 指令，以解析和链接文件之间的依赖关系。
    -   保留重要的元数据，如注释和源代码位置（可配置）。
    -   既可以从文件系统 (`NewParser`) 操作，也可以从内存中的文件映射 (`NewParserFromMap`) 操作。

---
### <a name="thriftwriter"></a> `thriftwriter/`

此包接收一个 `IDLSchema` (AST) 并生成格式化的 Thrift 源代码。

-   **功能**:
    -   从 AST 生成一个 `文件名 -> 内容` 的映射。
    -   产出语法正确且可读的 Thrift 代码。
    -   允许配置输出，例如选择包含或排除注释。

---
### <a name="thriftanalyzer"></a> `thriftanalyzer/`

为 Thrift 项目提供静态分析能力。

-   **功能**:
    -   构建一个详细的依赖图，显示文件之间是如何相互包含的。
    -   检测循环依赖，这可能导致代码生成问题。
    -   识别显式（两个文件为同一语言声明了相同的命名空间）和隐式（基于文件名）的 `namespace` 冲突。

---
### <a name="swagger2thrift"></a> `swagger2thrift/`

可将 REST API 规范翻译为 Thrift。

-   **功能**:
    -   兼容 OpenAPI v3 和 Swagger v2。
    -   管理从 JSON Schema到 Thrift `structs` 和 `enums` 的转换。
    -   将 API 端点转换为 Thrift `services` 和 `functions`，并添加注解（`api.get`, `api.post`）以保留 HTTP 路由信息。
    -   在处理多个规范文件时，自动将生成的 Thrift 代码组织到子目录中，以避免命名冲突。