package thriftanalyzer

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ToDOT 将 RichDependencyGraph 转换为 Graphviz DOT 格式的字符串。
// 这个字符串可以直接用于生成可视化图像。
func (g *RichDependencyGraph) ToDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph ThriftDependencies {\n")
	sb.WriteString("    rankdir=LR;\n") // 从左到右布局
	sb.WriteString("    node [shape=box, style=\"rounded,filled\", fontname=\"Helvetica\"];\n")
	sb.WriteString("    edge [fontname=\"Helvetica\"];\n\n")

	// 定义所有节点
	sb.WriteString("    // Nodes\n")
	for path, node := range g.Nodes {
		// 使用一个简短的 ID 作为节点名，避免路径中的特殊字符问题
		nodeID := sanitizeNodeID(path)

		// 节点的标签将显示更丰富的信息
		label := fmt.Sprintf("%s", node.BaseName)
		if len(node.Namespaces) > 0 {
			label += "\n---\n"
			var nsParts []string
			for _, ns := range node.Namespaces {
				nsParts = append(nsParts, fmt.Sprintf("%s: %s", ns.Scope, ns.Identifier))
			}
			// 使用 \l 来左对齐文本
			label += strings.Join(nsParts, "\n") + ``
		}

		// 为不同类型的节点设置不同的颜色
		color := "lightgrey"
		if node.IsEntryPoint {
			color = "lightblue"
		}
		if node.HasParseErrors {
			color = "lightpink"
		}

		sb.WriteString(fmt.Sprintf("    %s [label=\"%s\", fillcolor=%s];\n", nodeID, label, color))
	}
	sb.WriteString("\n")

	// 定义所有边
	sb.WriteString("    // Edges\n")
	for _, node := range g.Nodes {
		sourceID := sanitizeNodeID(node.AbsolutePath)
		for _, edge := range node.Includes {
			targetID := sanitizeNodeID(edge.TargetPath)

			// 边的样式
			style := "solid"
			edgeColor := "black"
			if edge.IsBroken {
				style = "dashed"
				edgeColor = "red"
			}

			sb.WriteString(fmt.Sprintf("    %s -> %s [style=%s, color=%s];\n",
				sourceID, targetID, style, edgeColor))
		}
	}

	sb.WriteString("}\n")

	return sb.String()
}

// sanitizeNodeID 将文件路径转换为合法的 DOT 节点 ID。
// (例如，替换 '/', '-', '.' 等字符)
func sanitizeNodeID(path string) string {
	r := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		".", "_",
		"-", "_",
	)
	return r.Replace(filepath.Clean(path))
}
