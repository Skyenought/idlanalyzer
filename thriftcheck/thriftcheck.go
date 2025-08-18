package thriftcheck

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/joyme123/protocol"
	"github.com/joyme123/thrift-ls/lsp/cache"
	"github.com/joyme123/thrift-ls/lsp/diagnostic"
	"go.lsp.dev/uri"
)

func ThriftSyntaxCheck(ctx context.Context, sources map[string][]byte) (map[string][]protocol.Diagnostic, error) {
	if len(sources) == 0 {
		return make(map[string][]protocol.Diagnostic), nil
	}

	fileChanges := make([]*cache.FileChange, 0, len(sources))
	fileURIs := make([]uri.URI, 0, len(sources))

	for filename, content := range sources {
		fileURI := uri.File(filename)
		change := &cache.FileChange{
			URI:     fileURI,
			Content: content,
			From:    cache.FileChangeTypeDidOpen, // Simulate opening the files.
		}
		fileChanges = append(fileChanges, change)
		fileURIs = append(fileURIs, fileURI)
	}

	snapshot := cache.BuildSnapshotForTest(fileChanges)

	allCheckers := []diagnostic.Interface{
		&diagnostic.Parse{},            // Checks for basic syntax errors.
		&diagnostic.CycleCheck{},       // Checks for circular include dependencies.
		&diagnostic.FieldIDCheck{},     // Checks for duplicate or invalid field IDs.
		&diagnostic.SemanticAnalysis{}, // The most powerful check: undefined types, name conflicts, etc.
	}

	allDiagnostics := make(map[string][]protocol.Diagnostic)
	var analysisErrors []string

	for _, checker := range allCheckers {
		result, err := checker.Diagnostic(ctx, snapshot, fileURIs)
		if err != nil {
			analysisErrors = append(analysisErrors, fmt.Sprintf("checker '%s' failed: %v", checker.Name(), err))
			continue
		}

		for u, diags := range result {
			filename := u.Filename()
			allDiagnostics[filename] = append(allDiagnostics[filename], diags...)
		}
	}

	var finalErr error
	if len(analysisErrors) > 0 {
		finalErr = errors.New(strings.Join(analysisErrors, "\n"))
	}

	return allDiagnostics, finalErr
}
