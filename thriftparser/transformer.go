package thriftparser

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/Skyenought/idlanalyzer/idl_ast"
	"github.com/joyme123/thrift-ls/lsp/cache"
	"github.com/joyme123/thrift-ls/lsp/codejump"
	"github.com/joyme123/thrift-ls/lsp/lsputils"
	"github.com/joyme123/thrift-ls/parser"
	"go.lsp.dev/uri"
)

// transformContext 用于在转换函数之间传递共享状态和信息
type transformContext struct {
	snapshot   *cache.Snapshot
	currentURI uri.URI
	currentAST *parser.Document
	rootDir    string
	source     []byte
	relPath    string
}

// transform 是主转换函数，将一个 thrift-ls 的 ParsedFile 转换为我们的 idl_ast.File
func transform(document *parser.Document, source []byte, fileURI uri.URI, rootDir string, snapshot *cache.Snapshot) (*idl_ast.File, error) {
	absPath := fileURI.Filename()
	relPath, err := filepath.Rel(rootDir, absPath)
	if err != nil {
		return nil, fmt.Errorf("could not compute relative path for %s: %w", absPath, err)
	}

	ctx := &transformContext{
		snapshot:   snapshot,
		currentURI: fileURI,
		currentAST: document,
		rootDir:    rootDir,
		source:     source,
		relPath:    relPath,
	}

	loc := convertLocation(document.Location)
	idlFile := &idl_ast.File{
		Path:       relPath,
		Location:   &loc,
		Imports:    transformImports(ctx),
		Namespaces: transformNamespaces(ctx),
		Definitions: idl_ast.Definitions{
			Services:  transformServices(ctx),
			Messages:  transformMessages(ctx),
			Enums:     transformEnums(ctx),
			Constants: transformConstants(ctx),
			Typedefs:  transformTypedefs(ctx),
		},
	}

	return idlFile, nil
}

func convertPosition(p parser.Position) idl_ast.Position {
	return idl_ast.Position{
		Line:   p.Line,
		Column: p.Col,
		Offset: p.Offset,
	}
}

func convertLocation(l parser.Location) idl_ast.Location {
	return idl_ast.Location{
		Start: convertPosition(l.StartPos),
		End:   convertPosition(l.EndPos),
	}
}

func convertComments(comments []*parser.Comment) []idl_ast.Comment {
	if comments == nil {
		return nil
	}
	res := make([]idl_ast.Comment, len(comments))
	for i, c := range comments {
		loc := convertLocation(c.Location)
		res[i] = idl_ast.Comment{
			Text:     c.Text,
			Location: &loc,
		}
	}
	return res
}

func isPrimitive(typeName string) bool {
	return codejump.IsBasicType(typeName)
}

// transformConstValue 将 parser.ConstValue 递归转换为 idl_ast.ConstantValue
func transformConstValue(cv *parser.ConstValue) *idl_ast.ConstantValue {
	if cv == nil {
		return nil
	}

	var val any
	switch cv.TypeName {
	case "string":
		if strVal, ok := cv.Value.(string); ok {
			val = fmt.Sprintf("%q", strVal)
		} else if literal, ok := cv.Value.(*parser.Literal); ok {
			val = fmt.Sprintf("%s%s%s", literal.Quote, literal.Value.Text, literal.Quote)
		}
	case "i64":
		val, _ = cv.Value.(int64)
	case "double":
		val, _ = cv.Value.(float64)
	case "identifier":
		strVal := cv.Value.(string)
		if bVal, err := strconv.ParseBool(strVal); err == nil {
			val = bVal
		} else {
			val = strVal
		}
	case "list":
		items := cv.Value.([]*parser.ConstValue)
		listVal := make([]*idl_ast.ConstantValue, 0, len(items))
		for _, item := range items {
			listVal = append(listVal, transformConstValue(item))
		}
		val = listVal
	case "map":
		items := cv.Value.([]*parser.ConstValue)
		mapVal := make([]*idl_ast.ConstantMapEntry, 0, len(items))
		for _, item := range items {
			if item.TypeName == "pair" {
				mapVal = append(mapVal, &idl_ast.ConstantMapEntry{
					Key:   transformConstValue(item.Key.(*parser.ConstValue)),
					Value: transformConstValue(item.Value.(*parser.ConstValue)),
				})
			}
		}
		val = mapVal
	}

	return &idl_ast.ConstantValue{Value: val}
}

// transformAnnotations 将 parser.Annotations 转换为 []idl_ast.Annotation
func transformAnnotations(p *parser.Annotations) []idl_ast.Annotation {
	if p == nil || len(p.Annotations) == 0 {
		return nil
	}
	res := make([]idl_ast.Annotation, len(p.Annotations))
	for i, anno := range p.Annotations {
		var constVal *idl_ast.ConstantValue
		if anno.Value != nil {
			tempConst := &parser.ConstValue{
				TypeName: "string",
				Value:    anno.Value,
			}
			constVal = transformConstValue(tempConst)
		}
		res[i] = idl_ast.Annotation{
			Name:  anno.Identifier.Name.Text,
			Value: constVal,
		}
	}
	return res
}

func transformImports(ctx *transformContext) []idl_ast.Import {
	imports := ctx.currentAST.Includes
	res := make([]idl_ast.Import, len(imports))
	for i, imp := range imports {
		absImportURI := lsputils.IncludeURI(ctx.currentURI, imp.Path.Value.Text)
		absImportPath := absImportURI.Filename()
		relPath, err := filepath.Rel(ctx.rootDir, absImportPath)
		if err != nil {
			relPath = absImportPath
		}
		originalValue := fmt.Sprintf("%s%s%s", imp.Path.Quote, imp.Path.Value.Text, imp.Path.Quote)
		loc := convertLocation(imp.Location)
		res[i] = idl_ast.Import{
			Comments: convertComments(imp.Comments),
			Location: &loc,
			Value:    originalValue,
			Path:     relPath,
		}
	}
	return res
}

func transformNamespaces(ctx *transformContext) []idl_ast.Namespace {
	namespaces := ctx.currentAST.Namespaces
	res := make([]idl_ast.Namespace, len(namespaces))
	for i, ns := range namespaces {
		loc := convertLocation(ns.Location)
		res[i] = idl_ast.Namespace{
			Comments: convertComments(ns.Comments),
			Location: &loc,
			Scope:    ns.Language.Name.Text,
			Name:     ns.Name.Name.Text,
		}
	}
	return res
}

func transformServices(ctx *transformContext) []idl_ast.Service {
	services := ctx.currentAST.Services
	res := make([]idl_ast.Service, len(services))
	for i, s := range services {
		startPos, endPos := getRealServicePositions(s)
		loc := convertLocation(parser.Location{StartPos: startPos, EndPos: endPos})
		extends := ""
		if s.Extends != nil {
			extends = s.Extends.Name.Text
		}
		name := s.Name.Name.Text
		res[i] = idl_ast.Service{
			Comments:           convertComments(s.Comments),
			Location:           &loc,
			Content:            getRealContent(ctx.source, startPos.Offset, endPos.Offset),
			Name:               name,
			FullyQualifiedName: fmt.Sprintf("%s#%s", ctx.relPath, name),
			Functions:          transformFunctions(s.Functions, name, ctx),
			Extends:            extends,
			Annotations:        transformAnnotations(s.Annotations),
		}
	}
	return res
}

func transformFunctions(functions []*parser.Function, serviceName string, ctx *transformContext) []idl_ast.Function {
	res := make([]idl_ast.Function, len(functions))
	for i, f := range functions {
		startPos, endPos := getRealFunctionPositions(f)
		loc := convertLocation(parser.Location{StartPos: startPos, EndPos: endPos})
		var returnType idl_ast.Type
		if f.Void != nil {
			voidLoc := convertLocation(f.Void.Location)
			returnType = idl_ast.Type{Location: &voidLoc, Name: "void", IsPrimitive: true}
		} else {
			returnType = transformType(f.FunctionType, ctx)
		}

		var throws []idl_ast.Field
		if f.Throws != nil {
			throws = transformFields(f.Throws.Fields, ctx)
		}

		sig, _ := getFunctionSignature(f, ctx.source)
		name := f.Name.Name.Text
		res[i] = idl_ast.Function{
			Comments:           convertComments(f.Comments),
			Location:           &loc,
			Signature:          sig,
			Name:               name,
			FullyQualifiedName: fmt.Sprintf("%s#%s.%s", ctx.relPath, serviceName, name),
			ReturnType:         returnType,
			Parameters:         transformFields(f.Arguments, ctx),
			Throws:             throws,
			Annotations:        transformAnnotations(f.Annotations),
		}
	}
	return res
}

func transformMessages(ctx *transformContext) []idl_ast.Message {
	structs := ctx.currentAST.Structs
	unions := ctx.currentAST.Unions
	exceptions := ctx.currentAST.Exceptions
	count := len(structs) + len(unions) + len(exceptions)
	res := make([]idl_ast.Message, 0, count)

	for _, s := range structs {
		startPos, endPos := getRealStructPositions(s)
		loc := convertLocation(parser.Location{StartPos: startPos, EndPos: endPos})
		name := s.Identifier.Name.Text
		res = append(res, idl_ast.Message{
			Comments:           convertComments(s.Comments),
			Location:           &loc,
			Content:            getRealContent(ctx.source, startPos.Offset, endPos.Offset),
			Name:               name,
			FullyQualifiedName: fmt.Sprintf("%s#%s", ctx.relPath, name),
			Type:               "struct",
			Fields:             transformFields(s.Fields, ctx),
			Annotations:        transformAnnotations(s.Annotations),
		})
	}
	for _, u := range unions {
		startPos, endPos := getRealUnionPositions(u)
		loc := convertLocation(parser.Location{StartPos: startPos, EndPos: endPos})
		name := u.Name.Name.Text
		res = append(res, idl_ast.Message{
			Comments:           convertComments(u.Comments),
			Location:           &loc,
			Content:            getRealContent(ctx.source, startPos.Offset, endPos.Offset),
			Name:               name,
			FullyQualifiedName: fmt.Sprintf("%s#%s", ctx.relPath, name),
			Type:               "union",
			Fields:             transformFields(u.Fields, ctx),
			Annotations:        transformAnnotations(u.Annotations),
		})
	}
	for _, e := range exceptions {
		startPos, endPos := getRealExceptionPositions(e)
		loc := convertLocation(parser.Location{StartPos: startPos, EndPos: endPos})
		name := e.Name.Name.Text
		res = append(res, idl_ast.Message{
			Comments:           convertComments(e.Comments),
			Location:           &loc,
			Content:            getRealContent(ctx.source, startPos.Offset, endPos.Offset),
			Name:               name,
			FullyQualifiedName: fmt.Sprintf("%s#%s", ctx.relPath, name),
			Type:               "exception",
			Fields:             transformFields(e.Fields, ctx),
			Annotations:        transformAnnotations(e.Annotations),
		})
	}
	return res
}

func transformFields(fields []*parser.Field, ctx *transformContext) []idl_ast.Field {
	res := make([]idl_ast.Field, len(fields))
	for i, f := range fields {
		required := "optional"
		if f.RequiredKeyword != nil {
			required = f.RequiredKeyword.Literal.Text
		}
		loc := convertLocation(f.Location)
		res[i] = idl_ast.Field{
			Comments:     convertComments(f.Comments),
			Location:     &loc,
			ID:           f.Index.Value,
			Name:         f.Identifier.Name.Text,
			Type:         transformType(f.FieldType, ctx),
			Required:     required,
			DefaultValue: transformConstValue(f.ConstValue),
			Annotations:  transformAnnotations(f.Annotations),
		}
	}
	return res
}

func transformEnums(ctx *transformContext) []idl_ast.Enum {
	enums := ctx.currentAST.Enums
	res := make([]idl_ast.Enum, len(enums))
	for i, e := range enums {
		startPos, endPos := getRealEnumPositions(e)
		loc := convertLocation(parser.Location{StartPos: startPos, EndPos: endPos})
		name := e.Name.Name.Text
		res[i] = idl_ast.Enum{
			Comments:           convertComments(e.Comments),
			Location:           &loc,
			Content:            getRealContent(ctx.source, startPos.Offset, endPos.Offset),
			Name:               name,
			FullyQualifiedName: fmt.Sprintf("%s#%s", ctx.relPath, name),
			Values:             transformEnumValues(e.Values),
			Annotations:        transformAnnotations(e.Annotations),
		}
	}
	return res
}

func transformEnumValues(values []*parser.EnumValue) []idl_ast.EnumValue {
	res := make([]idl_ast.EnumValue, len(values))
	for i, v := range values {
		loc := convertLocation(v.Location)
		res[i] = idl_ast.EnumValue{
			Comments:    convertComments(v.Comments),
			Location:    &loc,
			Name:        v.Name.Name.Text,
			Value:       int(v.Value),
			Annotations: transformAnnotations(v.Annotations),
		}
	}
	return res
}

func transformConstants(ctx *transformContext) []idl_ast.Constant {
	constants := ctx.currentAST.Consts
	res := make([]idl_ast.Constant, len(constants))
	for i, c := range constants {
		startPos, endPos := getRealConstPositions(c)
		loc := convertLocation(parser.Location{StartPos: startPos, EndPos: endPos})
		name := c.Name.Name.Text
		res[i] = idl_ast.Constant{
			Comments:           convertComments(c.Comments),
			Location:           &loc,
			Content:            getRealContent(ctx.source, startPos.Offset, endPos.Offset),
			Name:               name,
			FullyQualifiedName: fmt.Sprintf("%s#%s", ctx.relPath, name),
			Type:               transformType(c.ConstType, ctx),
			Value:              c.Value.ValueInText,
			Annotations:        transformAnnotations(c.Annotations),
		}
	}
	return res
}

func transformTypedefs(ctx *transformContext) []idl_ast.Typedef {
	typedefs := ctx.currentAST.Typedefs
	res := make([]idl_ast.Typedef, len(typedefs))
	for i, t := range typedefs {
		startPos, endPos := getRealTypedefPositions(t)
		loc := convertLocation(parser.Location{StartPos: startPos, EndPos: endPos})
		res[i] = idl_ast.Typedef{
			Comments:    convertComments(t.Comments),
			Location:    &loc,
			Content:     getRealContent(ctx.source, startPos.Offset, endPos.Offset),
			Alias:       t.Alias.Name.Text,
			Type:        transformType(t.T, ctx),
			Annotations: transformAnnotations(t.Annotations),
		}
	}
	return res
}

func transformType(p *parser.FieldType, ctx *transformContext) idl_ast.Type {
	if p == nil {
		return idl_ast.Type{}
	}
	loc := convertLocation(p.Location)
	t := idl_ast.Type{
		Location:    &loc,
		Name:        p.TypeName.Name,
		IsPrimitive: isPrimitive(p.TypeName.Name),
	}

	if !t.IsPrimitive && !codejump.IsContainerType(t.Name) {
		defURI, defIdentifier, _, err := codejump.TypeNameDefinitionIdentifier(
			context.TODO(),
			ctx.snapshot,
			ctx.currentURI,
			ctx.currentAST,
			p.TypeName,
		)
		if err == nil && defIdentifier != nil {
			relPath, err := filepath.Rel(ctx.rootDir, defURI.Filename())
			if err == nil {
				t.FullyQualifiedName = fmt.Sprintf("%s#%s", relPath, defIdentifier.Name.Text)
			}
		}
	}

	switch t.Name {
	case "map":
		if p.KeyType != nil {
			keyType := transformType(p.KeyType, ctx)
			t.KeyType = &keyType
		}
		if p.ValueType != nil {
			valueType := transformType(p.ValueType, ctx)
			t.ValueType = &valueType
		}
	case "list", "set":
		if p.KeyType != nil {
			valueType := transformType(p.KeyType, ctx)
			t.ValueType = &valueType
		}
	}

	return t
}
