package swagger2thrift

import (
	"reflect"

	"github.com/Skyenought/idlanalyzer/idl_ast"
)

type OpenAPISpec struct {
	OpenAPI    string               `yaml:"openapi"`
	Info       OpenAPIInfo          `yaml:"info"`
	Paths      map[string]*PathItem `yaml:"paths"`
	Components *Components          `yaml:"components"`
}

type OpenAPIInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type Components struct {
	Schemas map[string]*Schema `yaml:"schemas"`
}

type PathItem struct {
	Get    *Operation `yaml:"get"`
	Put    *Operation `yaml:"put"`
	Post   *Operation `yaml:"post"`
	Delete *Operation `yaml:"delete"`
	Patch  *Operation `yaml:"patch"`
}

type Operation struct {
	Tags        []string             `yaml:"tags"`
	Summary     string               `yaml:"summary"`
	Description string               `yaml:"description"`
	OperationID string               `yaml:"operationId"`
	Parameters  []*Parameter         `yaml:"parameters"`
	RequestBody *RequestBody         `yaml:"requestBody"`
	Responses   map[string]*Response `yaml:"responses"`
}

type Parameter struct {
	Name        string  `yaml:"name"`
	In          string  `yaml:"in"`
	Description string  `yaml:"description"`
	Required    bool    `yaml:"required"`
	Schema      *Schema `yaml:"schema"`
}

type RequestBody struct {
	Description string                `yaml:"description"`
	Content     map[string]*MediaType `yaml:"content"`
	Required    bool                  `yaml:"required"`
}

type Response struct {
	Description string                `yaml:"description"`
	Content     map[string]*MediaType `yaml:"content"`
}

type MediaType struct {
	Schema *Schema `yaml:"schema"`
}

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
	XEnumVarNames        []string           `yaml:"x-enum-varnames"`
	AllOf                []*Schema          `yaml:"allOf"`
}

type SwaggerSpec struct {
	Swagger     string                      `yaml:"swagger"`
	Info        OpenAPIInfo                 `yaml:"info"`
	Paths       map[string]*SwaggerPathItem `yaml:"paths"`
	Definitions map[string]*Schema          `yaml:"definitions"`
}

type SwaggerPathItem struct {
	Get    *SwaggerOperation `yaml:"get"`
	Put    *SwaggerOperation `yaml:"put"`
	Post   *SwaggerOperation `yaml:"post"`
	Delete *SwaggerOperation `yaml:"delete"`
	Patch  *SwaggerOperation `yaml:"patch"`
}

type SwaggerOperation struct {
	Tags        []string                    `yaml:"tags"`
	Summary     string                      `yaml:"summary"`
	Description string                      `yaml:"description"`
	OperationID string                      `yaml:"operationId"`
	Parameters  []*SwaggerParameter         `yaml:"parameters"`
	Responses   map[string]*SwaggerResponse `yaml:"responses"`
}

type SwaggerParameter struct {
	Name          string   `yaml:"name"`
	In            string   `yaml:"in"`
	Description   string   `yaml:"description"`
	Required      bool     `yaml:"required"`
	Schema        *Schema  `yaml:"schema"`
	Type          string   `yaml:"type"`
	Format        string   `yaml:"format"`
	Items         *Schema  `yaml:"items"`
	Enum          []any    `yaml:"enum"`
	XEnumVarNames []string `yaml:"x-enum-varnames"`
}

type SwaggerResponse struct {
	Description string  `yaml:"description"`
	Schema      *Schema `yaml:"schema"`
}

type canonicalDef struct {
	originalNamespace string
	originalFullName  string
	fileName          string
	definition        interface{}
}

type Converter struct {
	filePath        string
	spec            any
	fileDefinitions map[string]*idl_ast.Definitions
	requestStructs  map[string][]idl_ast.Message
	cfg             *Config
	definitionsMap  map[string]*Schema
	canonicalDefs   map[string]canonicalDef
}

func definitionsAreEqual(def1, def2 interface{}) bool {
	v1 := reflect.ValueOf(def1).Elem()
	v2 := reflect.ValueOf(def2).Elem()

	t1 := v1.Type()
	t2 := v2.Type()

	if t1 != t2 {
		return false
	}

	copy1 := reflect.New(t1).Elem()
	copy2 := reflect.New(t2).Elem()
	copy1.Set(v1)
	copy2.Set(v2)

	fqnField1 := copy1.FieldByName("FullyQualifiedName")
	if fqnField1.IsValid() {
		fqnField1.SetString("")
	}
	fqnField2 := copy2.FieldByName("FullyQualifiedName")
	if fqnField2.IsValid() {
		fqnField2.SetString("")
	}

	return reflect.DeepEqual(copy1.Interface(), copy2.Interface())
}
