# package `thriftanalyzer`

## 概述

`thriftanalyzer` 包是一个为 Apache Thrift 项目设计的静态分析引擎。它通过操作**基于 `abcoder` `UniAST` 理念设计的 `idl_ast`**，对整个 Thrift 项目的依赖关系进行深度分析，旨在发现潜在的结构性问题。

## 主要特性

-   **依赖图构建**: 能够解析一个项目中的所有 `include` 关系，并构建一个完整的 `RichDependencyGraph`（丰富依赖图）。这个图清晰地展示了文件（`FileNode`）之间是如何相互依赖的。
-   **循环依赖检测**: 自动识别 Thrift 定义中非法的循环 `include`（例如，`a.thrift` 包含 `b.thrift`，同时 `b.thrift` 包含 `a.thrift`），帮助开发者避免难以排查的编译错误。
-   **命名空间冲突检测**:
    -   **显式冲突**: 发现多个文件为同一种目标语言定义了完全相同的 `namespace`。
    -   **隐式冲突**: Thrift 在导入时会使用文件名作为默认命名空间，该工具能检测到由此可能引发的冲突（例如，项目中有两个都名为 `base.thrift` 的文件）。
-   **可配置分析**: 允许通过选项自定义分析行为，例如指定要关注的 `namespace` 作用域（如 `go`, `java` 等）。

## 使用指南

核心功能通过 `AnalyzeThriftDependencies` 函数提供。它接收一个入口文件路径和包含所有项目文件的 map，返回一个依赖图和一份包含所有问题的分析报告（作为 `error`）。

```go
// 函数签名
func AnalyzeThriftDependencies(mainIdlPath string, files map[string][]byte, options ...Option) (*RichDependencyGraph, error)
```

### 示例代码
```go
package main

import (
	"fmt"
	"github.com/Skyenought/idlanalyzer/thriftanalyzer"
)

func main() {
	// 1. 准备项目文件
	files := map[string][]byte{
		"/app/main.thrift": []byte(`include "a.thrift"`),
		"/app/a.thrift":    []byte(`include "b.thrift"\nnamespace go "common.api"`),
		"/app/b.thrift":    []byte(`include "a.thrift"`), // 循环依赖
		"/app/c.thrift":    []byte(`namespace go "common.api"`), // 命名空间冲突
	}
	mainPath := "/app/main.thrift"

	// 2. 执行分析
	graph, err := thriftanalyzer.AnalyzeThriftDependencies(mainPath, files)

	// 3. 处理分析结果
	if err != nil {
		fmt.Println("分析发现以下问题:")
		if analysisResult, ok := err.(*thriftanalyzer.AnalysisResult); ok {
			// 打印详细的冲突和循环信息
			fmt.Println(analysisResult.Error())
		}
	} else {
		fmt.Println("分析完成，未发现结构性问题。")
	}

	// 依赖图总是可用的，无论是否有错误
	fmt.Printf("入口文件 '%s' 包含 %d 个直接依赖。\n", graph.EntryPointPath, len(graph.Nodes[mainPath].Includes))
}
```