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

	allGeneratedFiles := make(map[string][]byte)

	// Process every file found in the map.
	for fileName, fileContent := range specs {
		// The core conversion logic is now in an internal function that accepts the config
		schema, err := convertInternal(fileName, fileContent, cfg)
		if err != nil {
			// Return a more specific error message
			return nil, fmt.Errorf("failed to convert spec '%s' to AST: %w", fileName, err)
		}

		// Generate the final Thrift file(s) from the AST for the current spec
		generatedFiles, err := thriftwriter.Generate(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to generate thrift files from AST for spec '%s': %w", fileName, err)
		}

		// Merge the newly generated files into our aggregate map
		for genFile, genContent := range generatedFiles {
			if _, exists := allGeneratedFiles[genFile]; exists {
				// Handle potential filename collisions if necessary, for now we can overwrite
				// or return an error. For simplicity, we'll log a warning and overwrite.
				// fmt.Printf("Warning: Duplicate filename '%s' generated from '%s'. Overwriting.\n", genFile, fileName)
			}
			allGeneratedFiles[genFile] = genContent
		}
	}

	return allGeneratedFiles, nil
}
