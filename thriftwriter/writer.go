package thriftwriter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Skyenought/idlanalyzer/idl_ast"
)

type Options struct {
	NoComments bool
}

type Option func(o *Options)

func WithNoComments(noComments bool) Option {
	return func(o *Options) {
		o.NoComments = noComments
	}
}

func Generate(schema *idl_ast.IDLSchema, opts ...Option) (map[string][]byte, error) {
	if schema == nil {
		return nil, fmt.Errorf("input schema cannot be nil")
	}
	options := &Options{
		NoComments: false,
	}
	for _, opt := range opts {
		opt(options)
	}
	outputFiles := make(map[string][]byte)
	for _, fileAST := range schema.Files {
		writer := &thriftWriter{
			b:                &strings.Builder{},
			indentationLevel: 0,
			indentStr:        "    ",
			opts:             options,
		}
		writer.writeFileContent(&fileAST)
		outputFiles[fileAST.Path] = []byte(writer.b.String())
	}
	return outputFiles, nil
}

type thriftWriter struct {
	b                *strings.Builder
	indentationLevel int
	indentStr        string
	opts             *Options
	includeBasenames map[string]struct{}
}

func (w *thriftWriter) writeFileContent(file *idl_ast.File) {
	w.includeBasenames = make(map[string]struct{})
	for _, imp := range file.Imports {
		base := filepath.Base(imp.Path)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		w.includeBasenames[name] = struct{}{}
	}

	w.writeNamespaces(file.Namespaces)
	w.writeImports(file.Imports)
	w.writeDefinitions(&file.Definitions)
}

func (w *thriftWriter) writeComments(comments []idl_ast.Comment, useIndent bool) {
	if w.opts.NoComments || len(comments) == 0 {
		return
	}
	for _, comment := range comments {
		text := strings.TrimRight(comment.Text, "\r\n")
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if useIndent {
				w.writeLine(line)
			} else {
				w.b.WriteString(line)
				w.b.WriteString("\n")
			}
		}
	}
}

func (w *thriftWriter) formatConstantValue(cv *idl_ast.ConstantValue) string {
	if cv == nil || cv.Value == nil {
		return ""
	}
	switch v := cv.Value.(type) {
	case string:
		if (strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`)) || (strings.HasPrefix(v, `'`) && strings.HasSuffix(v, `'`)) {
			return v
		}
		return v
	case int64, float64, bool:
		return fmt.Sprintf("%v", v)
	case []*idl_ast.ConstantValue:
		var items []string
		for _, item := range v {
			items = append(items, w.formatConstantValue(item))
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	case []*idl_ast.ConstantMapEntry:
		var entries []string
		for _, entry := range v {
			keyStr := w.formatConstantValue(entry.Key)
			valStr := w.formatConstantValue(entry.Value)
			entries = append(entries, fmt.Sprintf("%s: %s", keyStr, valStr))
		}
		return fmt.Sprintf("{%s}", strings.Join(entries, ", "))
	default:
		return ""
	}
}

func (w *thriftWriter) formatAnnotations(annos []idl_ast.Annotation) string {
	if len(annos) == 0 {
		return ""
	}
	var parts []string
	for _, anno := range annos {
		valStr, err := anno.Value.StringValue()
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf(`%s = "%s"`, anno.Name, valStr))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
}

func (w *thriftWriter) writeNamespaces(namespaces []idl_ast.Namespace) {
	if len(namespaces) == 0 {
		return
	}
	for _, ns := range namespaces {
		w.writeComments(ns.Comments, false)
		w.writeLinef("namespace %s %s", ns.Scope, ns.Name)
	}
	w.writeLine("")
}

func (w *thriftWriter) writeImports(imports []idl_ast.Import) {
	if len(imports) == 0 {
		return
	}
	for _, imp := range imports {
		w.writeComments(imp.Comments, false)
		w.writeLinef("include %s", imp.Value)
	}
	w.writeLine("")
}

func (w *thriftWriter) writeDefinitions(defs *idl_ast.Definitions) {
	if len(defs.Constants) > 0 {
		for i, constant := range defs.Constants {
			w.writeConstant(&constant)
			if i < len(defs.Constants)-1 {
				w.writeLine("")
			}
		}
		w.writeLine("")
	}
	if len(defs.Typedefs) > 0 {
		for i, typedef := range defs.Typedefs {
			w.writeTypedef(&typedef)
			if i < len(defs.Typedefs)-1 {
				w.writeLine("")
			}
		}
		w.writeLine("")
	}
	if len(defs.Enums) > 0 {
		for i, enum := range defs.Enums {
			w.writeEnum(&enum)
			if i < len(defs.Enums)-1 {
				w.writeLine("")
			}
		}
		w.writeLine("")
	}
	if len(defs.Messages) > 0 {
		for i, msg := range defs.Messages {
			w.writeMessage(&msg)
			if i < len(defs.Messages)-1 {
				w.writeLine("")
			}
		}
		w.writeLine("")
	}
	if len(defs.Services) > 0 {
		for i, svc := range defs.Services {
			w.writeService(&svc)
			if i < len(defs.Services)-1 {
				w.writeLine("")
			}
		}
		w.writeLine("")
	}
}

func (w *thriftWriter) writeConstant(c *idl_ast.Constant) {
	w.writeComments(c.Comments, false)
	typeStr := w.formatType(&c.Type)
	line := fmt.Sprintf("const %s %s = %s%s", typeStr, c.Name, c.Value, w.formatAnnotations(c.Annotations))
	w.writeLine(line)
}

func (w *thriftWriter) writeTypedef(td *idl_ast.Typedef) {
	w.writeComments(td.Comments, false)
	typeStr := w.formatType(&td.Type)
	line := fmt.Sprintf("typedef %s %s%s", typeStr, td.Alias, w.formatAnnotations(td.Annotations))
	w.writeLine(line)
}

func (w *thriftWriter) writeEnum(e *idl_ast.Enum) {
	w.writeComments(e.Comments, false)
	line := fmt.Sprintf("enum %s%s {", e.Name, w.formatAnnotations(e.Annotations))
	w.writeLine(line)
	w.indent()
	for i, val := range e.Values {
		w.writeEnumValue(&val, i < len(e.Values)-1)
	}
	w.unindent()
	w.writeLine("}")
}

func (w *thriftWriter) writeEnumValue(val *idl_ast.EnumValue, needsComma bool) {
	w.writeComments(val.Comments, true)
	line := fmt.Sprintf("%s = %d%s", val.Name, val.Value, w.formatAnnotations(val.Annotations))
	if needsComma {
		line += ","
	}
	w.writeLine(line)
}

func (w *thriftWriter) writeMessage(m *idl_ast.Message) {
	w.writeComments(m.Comments, false)
	header := fmt.Sprintf("%s %s%s {", m.Type, m.Name, w.formatAnnotations(m.Annotations))
	w.writeLine(header)
	w.indent()
	for _, field := range m.Fields {
		w.writeField(&field, true)
	}
	w.unindent()
	w.writeLine("}")
}

func (w *thriftWriter) writeField(f *idl_ast.Field, trailingSeparator bool) {
	w.writeComments(f.Comments, true)
	var parts []string
	parts = append(parts, fmt.Sprintf("%d:", f.ID))
	if f.Required != "" && f.Required != "optional" {
		parts = append(parts, f.Required)
	}
	parts = append(parts, w.formatType(&f.Type))
	parts = append(parts, f.Name)
	if f.DefaultValue != nil {
		defaultValueStr := w.formatConstantValue(f.DefaultValue)
		parts = append(parts, "=", defaultValueStr)
	}
	line := strings.Join(parts, " ") + w.formatAnnotations(f.Annotations)
	if trailingSeparator {
		line += ","
	}
	w.writeLine(line)
}

func (w *thriftWriter) writeService(s *idl_ast.Service) {
	w.writeComments(s.Comments, false)
	header := fmt.Sprintf("service %s", s.Name)
	if s.Extends != "" {
		header += fmt.Sprintf(" extends %s", s.Extends)
	}
	header += w.formatAnnotations(s.Annotations)
	header += " {"
	w.writeLine(header)
	w.indent()
	for i, fun := range s.Functions {
		w.writeComments(fun.Comments, true)
		line := w.formatFunction(&fun)
		if i < len(s.Functions)-1 {
			line += ","
		}
		w.writeLine(line)
		if i < len(s.Functions)-1 {
			w.writeLine("")
		}
	}
	w.unindent()
	w.writeLine("}")
}

func (w *thriftWriter) formatFunction(f *idl_ast.Function) string {
	isOneway := f.ReturnType.Name == "void" && len(f.Throws) == 0
	onewayStr := ""
	if isOneway {
		onewayStr = "oneway "
	}
	returnTypeStr := w.formatType(&f.ReturnType)
	var paramParts []string
	for _, p := range f.Parameters {
		paramParts = append(paramParts, w.formatParamField(&p))
	}
	paramsStr := strings.Join(paramParts, ", ")
	var throwsParts []string
	if len(f.Throws) > 0 {
		for _, t := range f.Throws {
			throwsParts = append(throwsParts, w.formatParamField(&t))
		}
	}
	throwsStr := ""
	if len(throwsParts) > 0 {
		throwsStr = fmt.Sprintf(" throws (%s)", strings.Join(throwsParts, ", "))
	}
	return fmt.Sprintf("%s%s %s(%s)%s%s", onewayStr, returnTypeStr, f.Name, paramsStr, throwsStr, w.formatAnnotations(f.Annotations))
}

func (w *thriftWriter) formatParamField(f *idl_ast.Field) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("%d:", f.ID))
	paramName := f.Name
	if _, ok := w.includeBasenames[paramName]; ok {
		paramName += "_"
	}
	parts = append(parts, w.formatType(&f.Type))
	parts = append(parts, paramName)
	if f.DefaultValue != nil {
		parts = append(parts, "=", w.formatConstantValue(f.DefaultValue))
	}
	return strings.Join(parts, " ") + w.formatAnnotations(f.Annotations)
}

func (w *thriftWriter) formatType(t *idl_ast.Type) string {
	if t == nil {
		return ""
	}
	switch t.Name {
	case "map":
		return fmt.Sprintf("map<%s, %s>", w.formatType(t.KeyType), w.formatType(t.ValueType))
	case "list":
		return fmt.Sprintf("list<%s>", w.formatType(t.ValueType))
	case "set":
		return fmt.Sprintf("set<%s>", w.formatType(t.ValueType))
	default:
		return t.Name
	}
}

func (w *thriftWriter) indent() {
	w.indentationLevel++
}

func (w *thriftWriter) unindent() {
	if w.indentationLevel > 0 {
		w.indentationLevel--
	}
}

func (w *thriftWriter) getIndent() string {
	return strings.Repeat(w.indentStr, w.indentationLevel)
}

func (w *thriftWriter) writeLine(s string) {
	if s != "" {
		w.b.WriteString(w.getIndent())
		w.b.WriteString(s)
	}
	w.b.WriteString("\n")
}

func (w *thriftWriter) writeLinef(format string, a ...any) {
	w.writeLine(fmt.Sprintf(format, a...))
}
