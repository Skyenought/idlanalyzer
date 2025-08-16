package swagger2thrift

import "github.com/Skyenought/idlanalyzer/idl_ast"

// ... (OpenAPI 和 Swagger 的结构体定义保持不变) ...
// OpenAPISpec represents the root of an OpenAPI 3.0 document.
type OpenAPISpec struct {
	OpenAPI    string               `yaml:"openapi"`
	Info       OpenAPIInfo          `yaml:"info"`
	Paths      map[string]*PathItem `yaml:"paths"`
	Components *Components          `yaml:"components"`
}

// OpenAPIInfo provides metadata about the API.
type OpenAPIInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

// Components holds reusable objects for different aspects of the OAS.
type Components struct {
	Schemas map[string]*Schema `yaml:"schemas"`
}

// PathItem describes the operations available on a single path.
type PathItem struct {
	Get    *Operation `yaml:"get"`
	Put    *Operation `yaml:"put"`
	Post   *Operation `yaml:"post"`
	Delete *Operation `yaml:"delete"`
	Patch  *Operation `yaml:"patch"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Tags        []string             `yaml:"tags"`
	Summary     string               `yaml:"summary"`
	Description string               `yaml:"description"`
	OperationID string               `yaml:"operationId"`
	Parameters  []*Parameter         `yaml:"parameters"`
	RequestBody *RequestBody         `yaml:"requestBody"`
	Responses   map[string]*Response `yaml:"responses"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Name        string  `yaml:"name"`
	In          string  `yaml:"in"` // "query", "header", "path", "cookie"
	Description string  `yaml:"description"`
	Required    bool    `yaml:"required"`
	Schema      *Schema `yaml:"schema"`
}

// RequestBody describes a single request body.
type RequestBody struct {
	Description string                `yaml:"description"`
	Content     map[string]*MediaType `yaml:"content"`
	Required    bool                  `yaml:"required"`
}

// Response describes a single response from an API Operation.
type Response struct {
	Description string                `yaml:"description"`
	Content     map[string]*MediaType `yaml:"content"`
}

// MediaType provides schema and examples for the media type.
type MediaType struct {
	Schema *Schema `yaml:"schema"`
}

// Schema is the heart of the data model definition.
type Schema struct {
	Type                 string             `yaml:"type"`
	Format               string             `yaml:"format"`
	Ref                  string             `yaml:"$ref"`
	Description          string             `yaml:"description"`
	Properties           map[string]*Schema `yaml:"properties"`
	Required             []string           `yaml:"required"`
	Items                *Schema            `yaml:"items"`
	Enum                 []any              `yaml:"enum"`
	Default              any                `yaml:"default"`
	AdditionalProperties any                `yaml:"additionalProperties"`
	AllOf                []*Schema          `yaml:"allOf"` // 新增 AllOf 字段
}

// SwaggerSpec represents the root of a Swagger 2.0 document.
type SwaggerSpec struct {
	Swagger     string                      `yaml:"swagger"`
	Info        OpenAPIInfo                 `yaml:"info"`
	Paths       map[string]*SwaggerPathItem `yaml:"paths"`
	Definitions map[string]*Schema          `yaml:"definitions"`
}

// SwaggerPathItem describes the operations available on a single path in Swagger 2.0.
type SwaggerPathItem struct {
	Get    *SwaggerOperation `yaml:"get"`
	Put    *SwaggerOperation `yaml:"put"`
	Post   *SwaggerOperation `yaml:"post"`
	Delete *SwaggerOperation `yaml:"delete"`
	Patch  *SwaggerOperation `yaml:"patch"`
}

// SwaggerOperation describes a single API operation in Swagger 2.0.
type SwaggerOperation struct {
	Tags        []string                    `yaml:"tags"`
	Summary     string                      `yaml:"summary"`
	Description string                      `yaml:"description"`
	OperationID string                      `yaml:"operationId"`
	Parameters  []*SwaggerParameter         `yaml:"parameters"`
	Responses   map[string]*SwaggerResponse `yaml:"responses"`
}

// SwaggerParameter describes a single operation parameter in Swagger 2.0.
type SwaggerParameter struct {
	Name        string  `yaml:"name"`
	In          string  `yaml:"in"`
	Description string  `yaml:"description"`
	Required    bool    `yaml:"required"`
	Schema      *Schema `yaml:"schema"` // Used for "in: body"
	Type        string  `yaml:"type"`   // Used for other 'in' types
	Format      string  `yaml:"format"`
	Items       *Schema `yaml:"items"` // Used for array types
}

// SwaggerResponse describes a single response from an API Operation in Swagger 2.0.
type SwaggerResponse struct {
	Description string  `yaml:"description"`
	Schema      *Schema `yaml:"schema"`
}

// Converter holds the state and logic for the conversion process.
type Converter struct {
	filePath        string
	spec            any
	fileDefinitions map[string]*idl_ast.Definitions
	requestStructs  map[string][]idl_ast.Message
	cfg             *Config
	definitionsMap  map[string]*Schema // 新增，用于快速查找 $ref
}
