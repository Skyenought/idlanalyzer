package swagger2thrift

import (
	"fmt"
	"github.com/Skyenought/idlanalyzer/thriftwriter"
)

// ConvertSpecsToThrift converts a map of OpenAPI/Swagger specifications into a map of generated Thrift files.
// It processes the first specification found in the input map.
//
// Parameters:
//   - specs: A map where the key is the filename of the specification (e.g., "api.json")
//     and the value is the byte content of the file.
//   - options: A variable number of functional options to customize the conversion,
//     such as setting a custom namespace with WithNamespace().
//
// Returns:
//   - A map where keys are the generated Thrift filenames (e.g., "service.thrift")
//     and values are their byte content.
//   - An error if the conversion process fails at any stage.
func ConvertSpecsToThrift(specs map[string][]byte, options ...Option) (map[string][]byte, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("input specs map cannot be empty")
	}

	// Apply options to our configuration
	cfg := &Config{}
	for _, opt := range options {
		opt(cfg)
	}

	// Process the first file found in the map.
	// In a typical scenario, this map will only contain one spec.
	var fileName string
	var fileContent []byte
	for k, v := range specs {
		fileName = k
		fileContent = v
		break
	}

	// The core conversion logic is now in an internal function that accepts the config
	schema, err := convertInternal(fileName, fileContent, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert spec '%s' to AST: %w", fileName, err)
	}

	// Generate the final Thrift file(s) from the AST
	generatedFiles, err := thriftwriter.Generate(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate thrift files from AST: %w", err)
	}

	return generatedFiles, nil
}
