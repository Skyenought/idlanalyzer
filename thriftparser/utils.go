package thriftparser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Skyenought/idlanalyzer/idl_ast"
	"github.com/joyme123/thrift-ls/format"
	"github.com/joyme123/thrift-ls/lsp/cache"
	"github.com/joyme123/thrift-ls/lsp/memoize"
	"go.lsp.dev/uri"

	"github.com/joyme123/thrift-ls/parser"
	"github.com/joyme123/thrift-ls/utils"
)

func getRealFunctionPositions(fn *parser.Function) (parser.Position, parser.Position) {
	if fn == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}

	// 确定起始位置
	startPos := fn.Pos() // 默认使用节点自身的起始位置
	if fn.Oneway != nil {
		startPos = fn.Oneway.Pos()
	} else if fn.Void != nil {
		startPos = fn.Void.Pos()
	} else if fn.FunctionType != nil {
		startPos = fn.FunctionType.Pos()
	}

	// 确定结束位置
	endPos := fn.End() // 默认使用节点自身的结束位置
	if fn.ListSeparatorKeyword != nil {
		endPos = fn.ListSeparatorKeyword.End()
	} else if fn.Annotations != nil {
		endPos = fn.Annotations.End()
	} else if fn.Throws != nil {
		endPos = fn.Throws.End()
	} else if fn.RParKeyword != nil {
		endPos = fn.RParKeyword.End()
	}

	return startPos, endPos
}

func getRealContent(source []byte, startOffset, endOffset int) string {
	if source == nil {
		return ""
	}
	sourceLen := len(source)
	if startOffset < 0 || endOffset > sourceLen || startOffset > endOffset {
		fmt.Errorf("invalid content offset. Start: %d, End: %d, Source Length: %d", startOffset, endOffset, sourceLen)
		return ""
	}
	return string(source[startOffset:endOffset])
}

// getRealStructPositions returns a Struct node's real start and end positions (from 'struct' keyword to '}').
func getRealStructPositions(s *parser.Struct) (parser.Position, parser.Position) {
	if s == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := s.Location.StartPos
	if s.StructKeyword != nil {
		startPos = s.StructKeyword.Pos()
	}
	endPos := s.Location.EndPos
	if s.RCurKeyword != nil {
		endPos = s.RCurKeyword.End()
	}
	return startPos, endPos
}

// getRealEnumPositions returns an Enum node's real start and end positions.
func getRealEnumPositions(e *parser.Enum) (parser.Position, parser.Position) {
	if e == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := e.Location.StartPos
	if e.EnumKeyword != nil {
		startPos = e.EnumKeyword.Pos()
	}
	endPos := e.Location.EndPos
	if e.RCurKeyword != nil {
		endPos = e.RCurKeyword.End()
	}
	return startPos, endPos
}

// getRealServicePositions returns a Service node's real start and end positions.
func getRealServicePositions(s *parser.Service) (parser.Position, parser.Position) {
	if s == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := s.Location.StartPos
	if s.ServiceKeyword != nil {
		startPos = s.ServiceKeyword.Pos()
	}
	endPos := s.Location.EndPos
	if s.RCurKeyword != nil {
		endPos = s.RCurKeyword.End()
	}
	return startPos, endPos
}

// getRealExceptionPositions returns an Exception node's real start and end positions.
func getRealExceptionPositions(e *parser.Exception) (parser.Position, parser.Position) {
	if e == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := e.Location.StartPos
	if e.ExceptionKeyword != nil {
		startPos = e.ExceptionKeyword.Pos()
	}
	endPos := e.Location.EndPos
	if e.RCurKeyword != nil {
		endPos = e.RCurKeyword.End()
	}
	return startPos, endPos
}

// getRealUnionPositions returns a Union node's real start and end positions.
func getRealUnionPositions(u *parser.Union) (parser.Position, parser.Position) {
	if u == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := u.Location.StartPos
	if u.UnionKeyword != nil {
		startPos = u.UnionKeyword.Pos()
	}
	endPos := u.Location.EndPos
	if u.RCurKeyword != nil {
		endPos = u.RCurKeyword.End()
	}
	return startPos, endPos
}

// getRealTypedefPositions returns a Typedef node's real start and end positions.
func getRealTypedefPositions(t *parser.Typedef) (parser.Position, parser.Position) {
	if t == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := t.Location.StartPos
	if t.TypedefKeyword != nil {
		startPos = t.TypedefKeyword.Pos()
	}
	endPos := t.Location.EndPos
	if t.Alias != nil {
		endPos = t.Alias.End()
	}
	return startPos, endPos
}

// getRealConstPositions returns a Const definition's real start and end positions.
func getRealConstPositions(c *parser.Const) (parser.Position, parser.Position) {
	if c == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := c.Location.StartPos
	if c.ConstKeyword != nil {
		startPos = c.ConstKeyword.Pos()
	}
	endPos := c.Location.EndPos
	if c.ListSeparatorKeyword != nil {
		endPos = c.ListSeparatorKeyword.End()
	} else if c.Value != nil {
		endPos = c.Value.End()
	}
	return startPos, endPos
}

func getRealFunctionEndOffset(fn *parser.Function) int {
	if fn == nil {
		return -1
	}
	if fn.ListSeparatorKeyword != nil {
		return fn.ListSeparatorKeyword.End().Offset
	}
	if fn.Annotations != nil {
		return fn.Annotations.End().Offset
	}
	if fn.Throws != nil {
		return fn.Throws.End().Offset
	}
	if fn.RParKeyword != nil {
		return fn.RParKeyword.End().Offset
	}
	return fn.End().Offset
}

func getRealFunctionStartOffset(fn *parser.Function) int {
	if fn == nil {
		return -1
	}
	if fn.Oneway != nil {
		return fn.Oneway.Pos().Offset
	}
	if fn.Void != nil {
		return fn.Void.Pos().Offset
	}
	if fn.FunctionType != nil {
		return fn.FunctionType.Pos().Offset
	}
	return fn.Pos().Offset
}

// getFunctionSignature 提取完整的函数签名
func getFunctionSignature(function *parser.Function, source []byte) (string, error) {
	if function == nil || source == nil {
		return "", fmt.Errorf("function node or source is nil")
	}

	startOffset := getRealFunctionStartOffset(function)
	endOffset := getRealFunctionEndOffset(function)

	sourceLen := len(source)
	if startOffset < 0 || endOffset > sourceLen || startOffset > endOffset {
		return "", fmt.Errorf("invalid offset range for function '%s'. Start: %d, End: %d", function.Name.Name.Text, startOffset, endOffset)
	}

	signatureBytes := source[startOffset:endOffset]
	signature := strings.TrimSpace(string(signatureBytes))

	// Thrift 规范中，最后一个函数声明后可以有逗号或分号，我们将其去除
	if strings.HasSuffix(signature, ",") || strings.HasSuffix(signature, ";") {
		signature = signature[:len(signature)-1]
		signature = strings.TrimSpace(signature)
	}

	return signature, nil
}

func removeAllComments(doc *parser.Document) {
	if doc == nil {
		return
	}
	removeCommentsRecursive(doc)
}

// removeCommentsRecursive 是一个递归函数，用于遍历 AST 并清除注释。
func removeCommentsRecursive(node parser.Node) {
	if utils.IsNil(node) {
		return
	}

	switch n := node.(type) {
	case *parser.Document:
		n.Comments = nil

	case *parser.Include:
		n.Comments = nil
		n.EndLineComments = nil
		if n.IncludeKeyword != nil {
			n.IncludeKeyword.Comments = nil
		}
	case *parser.CPPInclude:
		n.Comments = nil
		n.EndLineComments = nil
		if n.CPPIncludeKeyword != nil {
			n.CPPIncludeKeyword.Comments = nil
		}
	case *parser.Namespace:
		n.Comments = nil
		n.EndLineComments = nil
		if n.NamespaceKeyword != nil {
			n.NamespaceKeyword.Comments = nil
		}

	case *parser.Struct:
		n.Comments = nil
		n.EndLineComments = nil
		if n.StructKeyword != nil {
			n.StructKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Union:
		n.Comments = nil
		n.EndLineComments = nil
		if n.UnionKeyword != nil {
			n.UnionKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Exception:
		n.Comments = nil
		n.EndLineComments = nil
		if n.ExceptionKeyword != nil {
			n.ExceptionKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Service:
		n.Comments = nil
		n.EndLineComments = nil
		if n.ServiceKeyword != nil {
			n.ServiceKeyword.Comments = nil
		}
		if n.ExtendsKeyword != nil {
			n.ExtendsKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Enum:
		n.Comments = nil
		n.EndLineComments = nil
		if n.EnumKeyword != nil {
			n.EnumKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Typedef:
		n.Comments = nil
		n.EndLineComments = nil
		if n.TypedefKeyword != nil {
			n.TypedefKeyword.Comments = nil
		}
	case *parser.Const:
		n.Comments = nil
		n.EndLineComments = nil
		if n.ConstKeyword != nil {
			n.ConstKeyword.Comments = nil
		}
		if n.EqualKeyword != nil {
			n.EqualKeyword.Comments = nil
		}
		if n.ListSeparatorKeyword != nil {
			n.ListSeparatorKeyword.Comments = nil
		}

	case *parser.Field:
		n.Comments = nil
		n.EndLineComments = nil
		if n.Index != nil {
			n.Index.Comments = nil
			if n.Index.ColonKeyword != nil {
				n.Index.ColonKeyword.Comments = nil
			}
		}
		if n.RequiredKeyword != nil {
			n.RequiredKeyword.Comments = nil
		}
		if n.EqualKeyword != nil {
			n.EqualKeyword.Comments = nil
		}
		if n.ListSeparatorKeyword != nil {
			n.ListSeparatorKeyword.Comments = nil
		}
	case *parser.Function:
		n.Comments = nil
		n.EndLineComments = nil
		if n.Oneway != nil {
			n.Oneway.Comments = nil
		}
		if n.Void != nil {
			n.Void.Comments = nil
		}
		if n.LParKeyword != nil {
			n.LParKeyword.Comments = nil
		}
		if n.RParKeyword != nil {
			n.RParKeyword.Comments = nil
		}
		if n.ListSeparatorKeyword != nil {
			n.ListSeparatorKeyword.Comments = nil
		}
	case *parser.EnumValue:
		n.Comments = nil
		n.EndLineComments = nil
		if n.EqualKeyword != nil {
			n.EqualKeyword.Comments = nil
		}
		if n.ListSeparatorKeyword != nil {
			n.ListSeparatorKeyword.Comments = nil
		}
	case *parser.Identifier:
		n.Comments = nil
	case *parser.Literal:
		n.Comments = nil
	case *parser.ConstValue:
		n.Comments = nil
	case *parser.FieldType:
		if n.TypeName != nil {
			n.TypeName.Comments = nil
		}
	case *parser.TypeName:
		n.Comments = nil
	}

	for _, child := range node.Children() {
		removeCommentsRecursive(child)
	}
}

// 文件: thriftparser/parser.go

// buildSnapshotWithMap 从内存中的文件 map 构建快照。
func (p *ThriftParser) buildSnapshotWithMap(name string, fileMap map[string][]byte) (*cache.Snapshot, []*cache.FileChange, error) {
	var fileChanges []*cache.FileChange

	for relativePath, content := range fileMap {
		if !strings.HasSuffix(relativePath, ".thrift") {
			continue
		}

		// 1. 构造一个逻辑上的绝对路径，用于 parser 和 URI
		// 使用 path/filepath.Join 可以正确处理路径分隔符
		logicalAbsPath := filepath.Join(p.rootDir, relativePath)

		// --- AST 解析逻辑不变 ---
		var finalAST *parser.Document
		// parser.Parse 的第一个参数 filename 主要用于错误信息中，使用逻辑路径即可
		initialAST, err := parser.Parse(logicalAbsPath, content)
		if err != nil {
			fmt.Errorf("Initial parse failed for %s: %v", logicalAbsPath, err)
			finalAST = nil
		} else {
			finalAST = initialAST.(*parser.Document)
		}

		if p.opts.NoComments && finalAST != nil {
			removeAllComments(finalAST)
			contentString, err := format.FormatDocument(finalAST)
			if err != nil {
				return nil, nil, fmt.Errorf("formatting after comment removal failed for %s: %w", logicalAbsPath, err)
			}
			content = []byte(contentString)

			reParsedAST, err := parser.Parse(logicalAbsPath, content)
			if err != nil {
				fmt.Errorf("re-parse after comment removal failed for %s: %v", logicalAbsPath, err)
				finalAST = nil
			} else {
				finalAST = reParsedAST.(*parser.Document)
			}
		}

		// 2. 使用这个逻辑绝对路径创建 URI
		// uri.File 会自动处理并生成 "file:///path/to/virtual/project/main.thrift" 格式
		uriFile := uri.File(logicalAbsPath)

		// p.fileAsts 的键是 URI 的 Filename() 部分，也就是逻辑绝对路径
		p.fileAsts[uriFile.Filename()] = finalAST

		fileChanges = append(fileChanges, &cache.FileChange{
			URI:     uriFile,
			Content: content,
			From:    cache.FileChangeTypeDidOpen,
		})
	}

	// --- 快照构建逻辑不变 ---
	store := &memoize.Store{}
	c := cache.New(store)
	fs := cache.NewOverlayFS(c)
	fs.Update(context.TODO(), fileChanges)

	// 使用 p.rootDir 构造 View 的根 URI
	view := cache.NewView(name, uri.File(p.rootDir), fs, store)
	ss := cache.NewSnapshot(view, store)

	for _, f := range fileChanges {
		ss.Parse(context.TODO(), f.URI)
	}

	return ss, fileChanges, nil
}

func (p *ThriftParser) buildSnapshot(name, folder string) (*cache.Snapshot, []*cache.FileChange, error) {
	var fileChanges []*cache.FileChange

	err := filepath.WalkDir(folder, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if !d.IsDir() && strings.HasSuffix(path, ".thrift") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			absPath, _ := filepath.Abs(path)

			var finalAST *parser.Document
			initialAST, err := parser.Parse(absPath, content)
			if err != nil {
				fmt.Errorf("Initial parse failed for %s: %v", absPath, err)
				finalAST = nil // Continue even if parsing fails.
			} else {
				finalAST = initialAST.(*parser.Document)
			}

			if p.opts.NoComments && finalAST != nil {
				removeAllComments(finalAST)
				contentString, err := format.FormatDocument(finalAST)
				if err != nil {
					return err // Formatting failure is a critical error.
				}
				content = []byte(contentString)

				reParsedAST, err := parser.Parse(absPath, content)
				if err != nil {
					fmt.Errorf("re-parse after comment removal failed for %s: %v", absPath, err)
					finalAST = nil
				} else {
					finalAST = reParsedAST.(*parser.Document)
				}
			}
			uriFile := uri.File(absPath)
			p.fileAsts[uriFile.Filename()] = finalAST

			fileChanges = append(fileChanges, &cache.FileChange{
				URI:     uriFile,
				Content: content,
				From:    cache.FileChangeTypeDidOpen,
			})
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("walk dir '%s' fail: %w", folder, err)
	}
	store := &memoize.Store{}
	c := cache.New(store)
	fs := cache.NewOverlayFS(c)
	fs.Update(context.TODO(), fileChanges)

	view := cache.NewView(name, uri.URI(fmt.Sprintf("file://%s", folder)), fs, store)
	ss := cache.NewSnapshot(view, store)

	for _, f := range fileChanges {
		ss.Parse(context.TODO(), f.URI)
	}

	return ss, fileChanges, nil
}

func removeLocationsInSchema(schema *idl_ast.IDLSchema) {
	if schema == nil {
		return
	}
	for i := range schema.Files {
		removeLocationsInFile(&schema.Files[i])
	}
}

func removeLocationsInFile(file *idl_ast.File) {
	if file == nil {
		return
	}
	file.Location = nil
	for i := range file.Imports {
		file.Imports[i].Location = nil
	}
	for i := range file.Namespaces {
		file.Namespaces[i].Location = nil
	}
	removeLocationsInDefinitions(&file.Definitions)
}

func removeLocationsInDefinitions(defs *idl_ast.Definitions) {
	for i := range defs.Services {
		removeLocationsInService(&defs.Services[i])
	}
	for i := range defs.Messages {
		removeLocationsInMessage(&defs.Messages[i])
	}
	for i := range defs.Enums {
		removeLocationsInEnum(&defs.Enums[i])
	}
	for i := range defs.Constants {
		defs.Constants[i].Location = nil
		removeLocationsInType(&defs.Constants[i].Type)
	}
	for i := range defs.Typedefs {
		removeLocationsInTypedef(&defs.Typedefs[i])
	}
}

func removeLocationsInService(svc *idl_ast.Service) {
	svc.Location = nil
	for i := range svc.Functions {
		removeLocationsInFunction(&svc.Functions[i])
	}
}

func removeLocationsInFunction(fun *idl_ast.Function) {
	fun.Location = nil
	removeLocationsInType(&fun.ReturnType)
	for i := range fun.Parameters {
		removeLocationsInField(&fun.Parameters[i])
	}
	for i := range fun.Throws {
		removeLocationsInField(&fun.Throws[i])
	}
}

func removeLocationsInMessage(msg *idl_ast.Message) {
	msg.Location = nil
	for i := range msg.Fields {
		removeLocationsInField(&msg.Fields[i])
	}
}

func removeLocationsInEnum(enum *idl_ast.Enum) {
	enum.Location = nil
	for i := range enum.Values {
		enum.Values[i].Location = nil
	}
}

func removeLocationsInTypedef(td *idl_ast.Typedef) {
	td.Location = nil
	removeLocationsInType(&td.Type)
}

func removeLocationsInField(field *idl_ast.Field) {
	field.Location = nil
	removeLocationsInType(&field.Type)
}

func removeLocationsInType(t *idl_ast.Type) {
	if t == nil {
		return
	}
	t.Location = nil
	if t.KeyType != nil {
		removeLocationsInType(t.KeyType)
	}
	if t.ValueType != nil {
		removeLocationsInType(t.ValueType)
	}
}
