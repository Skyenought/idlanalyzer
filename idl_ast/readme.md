# package `idl_ast`

## 概述

`idl_ast` (IDL Abstract Syntax Tree) 包是 IDL 分析器工具套件的基石。它定义了一系列 Go 结构体，共同构成了一个通用的抽象语法树（AST）。

**该 AST 的设计是 `abcoder` 框架中 `UniAST` 核心思想的一个具体实践**，专门为表示接口定义语言（IDL），特别是 Apache Thrift，提供了一种标准化的、语言无关的内存表示。

## 目的

通过提供一个统一的中间表示，`idl_ast` 将工具套件中的各个组件（解析器、分析器、写入器）与具体的 IDL 语法解耦。这使得所有工具都可以操作同一套标准化的数据结构，极大地提高了代码的可维护性和扩展性，完全符合 `abcoder` 的设计哲学。

## 关键结构体

-   `IDLSchema`: 代表一个完整的 IDL 项目的根对象，包含一个或多个 `File`。
-   `File`: 代表一个独立的 IDL 文件，包含了它的 `imports`, `namespaces` 和 `Definitions`。
-   `Definitions`: 一个容器，用于组织文件内的所有核心定义，如 `Services`, `Messages`, `Enums` 等。
-   `Service`, `Message`, `Enum`: 分别代表 IDL 中的服务、结构化数据类型（struct/union/exception）和枚举。
-   `Type`: 一个能够递归表示任意数据类型（从基本类型到复杂容器）的结构。
-   `search_ast.go`: 为 `IDLSchema` 提供了高效的查询方法，如 `FindServicesByFQN`，允许通过名称快速在整个项目中定位定义。

## 与 `abcoder` 的关系

`idl_ast` 可以被视为 `abcoder` `UniAST` 规范在 IDL 领域的一个专注实现。它验证了通过统一 AST 来处理代码的可行性，并为 `abcoder` 框架中更通用的语言处理能力奠定了基础。

## 用法

此包是连接工具套件其他部分的核心枢纽：

1.  **`thriftparser`** 或 **`swagger2thrift`** 读取源文件，并生成一个 `*idl_ast.IDLSchema` 实例。
2.  **`thriftanalyzer`** 或其他自定义工具，基于这个 `IDLSchema` 实例进行分析或修改。
3.  **`thriftwriter`** 接收这个（可能已被修改的） `IDLSchema` 实例，并将其转换回 `.thrift` 源代码。

## 具体规范

![desc.md](./decs.md)