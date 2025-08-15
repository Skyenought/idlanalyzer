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

	baseName := filepath.Base(c.filePath)
	namespaceName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	sanitizer := strings.NewReplacer("-", "_", ".", "_", " ", "_")
	namespaceName = sanitizer.Replace(namespaceName)

	for _, fileName := range fileNames {
		defs := c.fileDefinitions[fileName]
		file := idl_ast.File{
			Path:        fileName,
			Syntax:      syntax,
			Definitions: *defs,
			Namespaces: []idl_ast.Namespace{
				{Scope: "go", Name: namespaceName},
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
