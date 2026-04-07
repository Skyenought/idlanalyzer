package thriftparser

import (
	"github.com/Skyenought/idlanalyzer/idl_ast"
)

func SortSchema(schema *idl_ast.IDLSchema) {
	for i := range schema.Files {
		sortFileDefinitions(&schema.Files[i].Definitions)
	}
}

// sortFileDefinitions 对单个文件中的定义执行拓扑排序。
func sortFileDefinitions(defs *idl_ast.Definitions) {
	allDefs := make(map[string]interface{})
	var allDefNames []string

	addDef := func(name string, def interface{}) {
		if _, exists := allDefs[name]; !exists {
			allDefs[name] = def
			allDefNames = append(allDefNames, name)
		}
	}

	for _, m := range defs.Messages {
		addDef(m.Name, m)
	}
	for _, e := range defs.Enums {
		addDef(e.Name, e)
	}
	for _, t := range defs.Typedefs {
		addDef(t.Alias, t)
	}
	for _, c := range defs.Constants {
		addDef(c.Name, c)
	}
	for _, s := range defs.Services {
		addDef(s.Name, s)
	}

	// graph: key 依赖于 value 列表中的项 (例如, A 依赖 B -> graph[A] = [B])
	// reverseGraph: key 被 value 列表中的项所依赖 (例如, B 被 A 使用 -> reverseGraph[B] = [A])
	graph := make(map[string][]string)
	reverseGraph := make(map[string][]string)
	inDegree := make(map[string]int)

	for name, def := range allDefs {
		dependencies := extractDependencies(def)
		inDegree[name] = len(dependencies)
		graph[name] = dependencies
		for _, depName := range dependencies {
			reverseGraph[depName] = append(reverseGraph[depName], name)
		}
	}

	queue := make([]string, 0)
	for name := range allDefs {
		if inDegree[name] == 0 {
			queue = append(queue, name)
		}
	}

	var sortedNames []string
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		sortedNames = append(sortedNames, name)

		for _, dependent := range reverseGraph[name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// 如果存在循环依赖，排序将不完整。我们将继续处理已排序的部分。
	if len(sortedNames) < len(allDefNames) {
		// 出现循环依赖时，为避免数据丢失，添加剩余的节点，尽管它们的顺序无法保证。
		sortedSet := make(map[string]struct{}, len(sortedNames))
		for _, name := range sortedNames {
			sortedSet[name] = struct{}{}
		}
		for _, name := range allDefNames {
			if _, ok := sortedSet[name]; !ok {
				sortedNames = append(sortedNames, name)
			}
		}
	}

	newDefs := idl_ast.Definitions{}
	for _, name := range sortedNames {
		def := allDefs[name]
		switch v := def.(type) {
		case idl_ast.Message:
			newDefs.Messages = append(newDefs.Messages, v)
		case idl_ast.Enum:
			newDefs.Enums = append(newDefs.Enums, v)
		case idl_ast.Typedef:
			newDefs.Typedefs = append(newDefs.Typedefs, v)
		case idl_ast.Constant:
			newDefs.Constants = append(newDefs.Constants, v)
		case idl_ast.Service:
			newDefs.Services = append(newDefs.Services, v)
		}
	}

	*defs = newDefs
}

// extractDependencies 查找给定定义所依赖的所有类型名称。
func extractDependencies(def interface{}) []string {
	depSet := make(map[string]struct{})
	addDep := func(name string) {
		// 我们只关心同一文件内的依赖，所以不添加带限定符的名称。
		if name != "" && !isPrimitive(name) && name != "list" && name != "map" && name != "set" && name != "void" {
			depSet[name] = struct{}{}
		}
	}

	switch v := def.(type) {
	case idl_ast.Message:
		for _, field := range v.Fields {
			extractTypeDependencies(&field.Type, addDep)
		}
	case idl_ast.Typedef:
		extractTypeDependencies(&v.Type, addDep)
	case idl_ast.Constant:
		extractTypeDependencies(&v.Type, addDep)
	case idl_ast.Service:
		for _, fun := range v.Functions {
			extractTypeDependencies(&fun.ReturnType, addDep)
			for _, param := range fun.Parameters {
				extractTypeDependencies(&param.Type, addDep)
			}
			for _, throw := range fun.Throws {
				extractTypeDependencies(&throw.Type, addDep)
			}
		}
	}

	deps := make([]string, 0, len(depSet))
	for dep := range depSet {
		deps = append(deps, dep)
	}
	return deps
}

// extractTypeDependencies 是一个递归辅助函数，用于从 Type 结构中提取所有用户定义的类型名称。
func extractTypeDependencies(t *idl_ast.Type, addDep func(string)) {
	if t == nil {
		return
	}
	addDep(t.Name)
	if t.KeyType != nil {
		extractTypeDependencies(t.KeyType, addDep)
	}
	if t.ValueType != nil {
		extractTypeDependencies(t.ValueType, addDep)
	}
}
