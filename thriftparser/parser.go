package thriftparser

import (
	"fmt"
	"path/filepath"

	"github.com/joyme123/thrift-ls/parser"

	"github.com/Skyenought/idlanalyzer/idl_ast"
	"github.com/joyme123/thrift-ls/lsp/cache"
)

type Options struct {
	NoLocation bool
	NoComments bool
}

type Option func(*Options)

func WithNoLocation(noLocation bool) Option {
	return func(o *Options) {
		o.NoLocation = noLocation
	}
}

func WithNoComments(noComments bool) Option {
	return func(o *Options) {
		o.NoComments = noComments
	}
}

type ThriftParser struct {
	rootDir     string
	opts        *Options
	snapshot    *cache.Snapshot
	files       []*cache.FileChange
	relationMap map[string][]byte
	fileAsts    map[string]*parser.Document
	schema      *idl_ast.IDLSchema
}

func NewParser(rootDir string, opts ...Option) (tt *ThriftParser, err error) {
	if !filepath.IsAbs(rootDir) {
		rootDir, err = filepath.Abs(rootDir)
		if err != nil {
			return nil, err
		}
	}
	defaultOptions := &Options{
		NoLocation: false,
		NoComments: false,
	}

	for _, opt := range opts {
		opt(defaultOptions)
	}

	t := &ThriftParser{
		rootDir:     rootDir,
		relationMap: make(map[string][]byte),
		fileAsts:    make(map[string]*parser.Document),
		opts:        defaultOptions,
	}

	snapshot, files, err := t.buildSnapshot(rootDir, rootDir)
	if err != nil {
		return nil, err
	}
	t.snapshot = snapshot
	t.files = files

	return t, nil
}

func (p *ThriftParser) ParseIDLs() (*idl_ast.IDLSchema, error) {
	if p.schema != nil {
		return p.schema, nil
	}

	schema := &idl_ast.IDLSchema{
		SchemaVersion: "1.0",
		IDLType:       "thrift",
		Files:         make([]idl_ast.File, 0, len(p.files)),
	}

	for _, fileChange := range p.files {
		parsedFile, ok := p.fileAsts[fileChange.URI.Filename()]
		if !ok {
			return nil, fmt.Errorf("failed to get parsed file for %s", fileChange.URI.Filename())
		}

		idlFile, err := transform(parsedFile, fileChange.Content, fileChange.URI, p.rootDir, p.snapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to transform ast for %s: %w", fileChange.URI.Filename(), err)
		}
		if idlFile != nil {
			schema.Files = append(schema.Files, *idlFile)
		}
	}

	if p.opts.NoLocation {
		removeLocationsInSchema(schema)
	}

	p.schema = schema
	return p.schema, nil
}

func NewParserFromMap(rootDir string, fileMap map[string][]byte, opts ...Option) (*ThriftParser, error) {
	// 确保 rootDir 是一个干净的、类似 Unix 的绝对路径格式，以便于 URI 构建
	if !filepath.IsAbs(rootDir) {
		// 如果不是绝对路径，我们假设它是相对于根的，并为其添加前缀
		// 这样做可以避免很多路径问题
		rootDir = "/" + rootDir
	}
	// 清理路径，例如将 "a//b" 变为 "a/b"
	rootDir = filepath.Clean(rootDir)

	defaultOptions := &Options{
		NoLocation: false,
		NoComments: false,
	}
	for _, opt := range opts {
		opt(defaultOptions)
	}

	t := &ThriftParser{
		rootDir:     rootDir, // 直接使用传入的（清理过的）rootDir
		relationMap: make(map[string][]byte),
		fileAsts:    make(map[string]*parser.Document),
		opts:        defaultOptions,
	}

	// 调用基于 map 的快照构建函数
	// 将 "name" 参数也设置为 rootDir
	snapshot, files, err := t.buildSnapshotWithMap(rootDir, fileMap)
	if err != nil {
		return nil, err
	}
	t.snapshot = snapshot
	t.files = files

	return t, nil
}
