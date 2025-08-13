package idl_ast

import (
	"strings"
	"sync"
)

type definitionIndex struct {
	// a map from FullyQualifiedName to the actual definition object.
	// The object is stored as an `any` type to hold different definition types
	// (e.g., *Service, *Message, *Enum, *Constant, *Function).
	fqnMap map[string]any
}

var (
	indexOnce sync.Once
	index     *definitionIndex
)

func (schema *IDLSchema) buildIndex() {
	indexOnce.Do(func() {
		index = &definitionIndex{
			fqnMap: make(map[string]any),
		}
		for i := range schema.Files {
			file := &schema.Files[i] // Use pointer to avoid copying
			defs := &file.Definitions

			// 索引 Services 和它们的 Functions
			for j := range defs.Services {
				service := &defs.Services[j]
				if service.FullyQualifiedName != "" {
					index.fqnMap[service.FullyQualifiedName] = service
				}
				for k := range service.Functions {
					function := &service.Functions[k]
					if function.FullyQualifiedName != "" {
						index.fqnMap[function.FullyQualifiedName] = function
					}
				}
			}

			// 索引 Messages (structs, unions, exceptions)
			for j := range defs.Messages {
				message := &defs.Messages[j]
				if message.FullyQualifiedName != "" {
					index.fqnMap[message.FullyQualifiedName] = message
				}
			}

			// 索引 Enums
			for j := range defs.Enums {
				enum := &defs.Enums[j]
				if enum.FullyQualifiedName != "" {
					index.fqnMap[enum.FullyQualifiedName] = enum
				}
			}

			// 索引 Constants
			for j := range defs.Constants {
				constant := &defs.Constants[j]
				if constant.FullyQualifiedName != "" {
					index.fqnMap[constant.FullyQualifiedName] = constant
				}
			}
		}
	})
}

func (schema *IDLSchema) FindByFQN(fqn string) []any {
	schema.buildIndex()

	// 1. 尝试精确匹配
	if def, found := index.fqnMap[fqn]; found {
		return []any{def}
	}

	// 2. 如果精确匹配失败，则进行后缀匹配
	var results []any
	for key, def := range index.fqnMap {
		if strings.HasSuffix(key, "#"+fqn) || key == fqn {
			// 后缀匹配时，确保 # 前面的部分也匹配，或者 fqn 本身就是一个完整的后缀
			results = append(results, def)
		}
	}

	return results
}

// findByType is a generic helper to find definitions of a specific type.
func findByType[T any](schema *IDLSchema, fqn string) []T {
	results := schema.FindByFQN(fqn)
	var typedResults []T

	for _, result := range results {
		if typedResult, ok := result.(T); ok {
			typedResults = append(typedResults, typedResult)
		}
	}
	return typedResults
}

// FindServicesByFQN 查找所有 FQN 以指定字符串结尾的 Service。
func (schema *IDLSchema) FindServicesByFQN(fqn string) []*Service {
	return findByType[*Service](schema, fqn)
}

// FindMessagesByFQN 查找所有 FQN 以指定字符串结尾的 Message。
func (schema *IDLSchema) FindMessagesByFQN(fqn string) []*Message {
	return findByType[*Message](schema, fqn)
}

// FindEnumsByFQN 查找所有 FQN 以指定字符串结尾的 Enum。
func (schema *IDLSchema) FindEnumsByFQN(fqn string) []*Enum {
	return findByType[*Enum](schema, fqn)
}

// FindConstantsByFQN 查找所有 FQN 以指定字符串结尾的 Constant。
func (schema *IDLSchema) FindConstantsByFQN(fqn string) []*Constant {
	return findByType[*Constant](schema, fqn)
}

// FindFunctionsByFQN 查找所有 FQN 以指定字符串结尾的 Function。
func (schema *IDLSchema) FindFunctionsByFQN(fqn string) []*Function {
	return findByType[*Function](schema, fqn)
}

// --- 新增：更细粒度的查找函数 ---

// FindStructsByFQN 查找所有 FQN 以指定字符串结尾的 Struct 类型的 Message。
func (schema *IDLSchema) FindStructsByFQN(fqn string) []*Message {
	messages := schema.FindMessagesByFQN(fqn)
	var structs []*Message
	for _, msg := range messages {
		if msg.Type == "struct" {
			structs = append(structs, msg)
		}
	}
	return structs
}

// FindUnionsByFQN 查找所有 FQN 以指定字符串结尾的 Union 类型的 Message。
func (schema *IDLSchema) FindUnionsByFQN(fqn string) []*Message {
	messages := schema.FindMessagesByFQN(fqn)
	var unions []*Message
	for _, msg := range messages {
		if msg.Type == "union" {
			unions = append(unions, msg)
		}
	}
	return unions
}

// FindExceptionsByFQN 查找所有 FQN 以指定字符串结尾的 Exception 类型的 Message。
func (schema *IDLSchema) FindExceptionsByFQN(fqn string) []*Message {
	messages := schema.FindMessagesByFQN(fqn)
	var exceptions []*Message
	for _, msg := range messages {
		if msg.Type == "exception" {
			exceptions = append(exceptions, msg)
		}
	}
	return exceptions
}

// SplitFQN 是一个辅助函数，用于从 FQN 中解析出文件路径和定义名称。
// 例如："path/to/file.thrift#MyStruct" -> "path/to/file.thrift", "MyStruct", true
// 例如："path/to/file.thrift#MyService.myFunction" -> "path/to/file.thrift", "MyService.myFunction", true
func SplitFQN(fqn string) (filePath, definitionName string, ok bool) {
	parts := strings.SplitN(fqn, "#", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
