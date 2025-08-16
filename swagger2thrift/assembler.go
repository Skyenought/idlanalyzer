package swagger2thrift

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Skyenought/idlanalyzer/idl_ast"
)

func (c *Converter) assembleSchema(idlType, syntax string) *idl_ast.IDLSchema {
	var files []idl_ast.File

	fileNames := make([]string, 0, len(c.fileDefinitions))
	for name := range c.fileDefinitions {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)

	// 1. 计算 OpenAPI 文件基础命名空间，这将是所有生成命名空间的前缀。
	//    例如，对于 "docs_swagger_3.json"，这里会得到 "docs_swagger_3"。
	openapiBaseName := filepath.Base(c.filePath)
	openapiBaseNamespace := strings.TrimSuffix(openapiBaseName, filepath.Ext(openapiBaseName))
	sanitizer := strings.NewReplacer("-", "_", ".", "_", " ", "_")
	openapiBaseNamespace = sanitizer.Replace(openapiBaseNamespace)

	for _, fileName := range fileNames {
		defs := c.fileDefinitions[fileName]

		thriftFileOnly := filepath.Base(fileName)
		thriftFileBase := strings.TrimSuffix(thriftFileOnly, filepath.Ext(thriftFileOnly))
		sanitizedThriftFileBase := sanitizer.Replace(thriftFileBase)

		finalNamespace := fmt.Sprintf("%s.%s", openapiBaseNamespace, sanitizedThriftFileBase)

		file := idl_ast.File{
			Path:        fileName,
			Syntax:      syntax,
			Definitions: *defs,
			Namespaces: []idl_ast.Namespace{
				{Scope: "go", Name: finalNamespace},
			},
			Imports: c.calculateImports(fileName, defs),
		}
		files = append(files, file)
	}

	return &idl_ast.IDLSchema{
		SchemaVersion: "1.0",
		IDLType:       idlType,
		Files:         files,
	}
}

func (c *Converter) calculateImports(currentFilename string, defs *idl_ast.Definitions) []idl_ast.Import {
	neededNamespaces := make(map[string]struct{})
	currentFileNamespace := strings.TrimSuffix(currentFilename, ".thrift")

	for _, s := range defs.Services {
		for _, f := range s.Functions {
			c.collectNamespacesFromType(&f.ReturnType, currentFileNamespace, neededNamespaces)
			for i := range f.Parameters {
				c.collectNamespacesFromType(&f.Parameters[i].Type, currentFileNamespace, neededNamespaces)
			}
			for i := range f.Throws {
				c.collectNamespacesFromType(&f.Throws[i].Type, currentFileNamespace, neededNamespaces)
			}
		}
	}
	for _, m := range defs.Messages {
		for i := range m.Fields {
			c.collectNamespacesFromType(&m.Fields[i].Type, currentFileNamespace, neededNamespaces)
		}
	}
	for _, td := range defs.Typedefs {
		c.collectNamespacesFromType(&td.Type, currentFileNamespace, neededNamespaces)
	}

	var imports []idl_ast.Import
	if len(neededNamespaces) > 0 {
		sortedNamespaces := make([]string, 0, len(neededNamespaces))
		for ns := range neededNamespaces {
			sortedNamespaces = append(sortedNamespaces, ns)
		}
		sort.Strings(sortedNamespaces)
		for _, ns := range sortedNamespaces {
			path := ns + ".thrift"
			imports = append(imports, idl_ast.Import{
				Path:  path,
				Value: fmt.Sprintf(`"%s"`, path),
			})
		}
	}
	return imports
}

func (c *Converter) collectNamespacesFromType(t *idl_ast.Type, currentFileNamespace string, needed map[string]struct{}) {
	if t == nil {
		return
	}

	ns, _ := splitDefinitionName(t.Name)
	if ns != "main" && ns != currentFileNamespace && ns != "" {
		needed[ns] = struct{}{}
	}

	if t.KeyType != nil {
		c.collectNamespacesFromType(t.KeyType, currentFileNamespace, needed)
	}
	if t.ValueType != nil {
		c.collectNamespacesFromType(t.ValueType, currentFileNamespace, needed)
	}
}
