package thriftparser

import (
	"fmt"
	"path/filepath"

	"github.com/joyme123/thrift-ls/parser"

	"github.com/Skyenought/idlanalyzer/idl_ast"
	"github.com/joyme123/thrift-ls/lsp/cache"
)

type Options struct {
	NoLocation      bool
	NoComments      bool
	SortDefinitions bool // 新增选项，用于控制定义排序
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

// WithSortDefinitions 启用或禁用对每个文件内定义的拓扑排序。
// 当为 true 时，定义将被排序，以确保依赖项出现在使用它们的项之前。
func WithSortDefinitions(sort bool) Option {
	return func(o *Options) {
		o.SortDefinitions = sort
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
		NoLocation:      false,
		NoComments:      false,
		SortDefinitions: false,
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

	if p.opts.SortDefinitions {
		SortSchema(schema)
	}

	if p.opts.NoLocation {
		removeLocationsInSchema(schema)
	}

	p.schema = schema
	return p.schema, nil
}

func NewParserFromMap(rootDir string, fileMap map[string][]byte, opts ...Option) (*ThriftParser, error) {
	if !filepath.IsAbs(rootDir) {
		rootDir = "/" + rootDir
	}
	// 清理路径，例如将 "a//b" 变为 "a/b"
	rootDir = filepath.Clean(rootDir)

	defaultOptions := &Options{
		NoLocation:      false,
		NoComments:      false,
		SortDefinitions: false, // 默认值为 false
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

	snapshot, files, err := t.buildSnapshotWithMap(rootDir, fileMap)
	if err != nil {
		return nil, err
	}
	t.snapshot = snapshot
	t.files = files

	return t, nil
}
