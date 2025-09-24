package swagger2thrift

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Skyenought/idlanalyzer/idl_ast"
	"gopkg.in/yaml.v3"
)

// convertInternal is the main entry point. It detects the spec version and converts it.
// It now includes a fallback mechanism to handle specs missing a version field.
func convertInternal(filePath string, content []byte, cfg *Config) (*idl_ast.IDLSchema, error) {
	var genericSpec map[string]interface{}
	if err := yaml.Unmarshal(content, &genericSpec); err != nil {
		return nil, fmt.Errorf("failed to parse spec file: %w", err)
	}

	converter := &Converter{
		filePath:        filePath,
		fileDefinitions: make(map[string]*idl_ast.Definitions),
		requestStructs:  make(map[string][]idl_ast.Message),
		cfg:             cfg,
	}

	if swaggerVersion, ok := genericSpec["swagger"].(string); ok && strings.HasPrefix(swaggerVersion, "2.") {
		var spec SwaggerSpec
		if err := yaml.Unmarshal(content, &spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Swagger v2 spec: %w", err)
		}
		converter.spec = &spec
		return converter.convertV2()
	}

	if openAPIVersion, ok := genericSpec["openapi"].(string); ok && strings.HasPrefix(openAPIVersion, "3.") {
		var spec OpenAPISpec
		if err := yaml.Unmarshal(content, &spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OpenAPI v3 spec: %w", err)
		}
		converter.spec = &spec
		return converter.convertV3()
	}

	// 2. 进入兜底机制：如果版本字段缺失
	if _, hasPaths := genericSpec["paths"]; !hasPaths {
		return nil, fmt.Errorf("unsupported or missing 'swagger'/'openapi' version field, and no 'paths' field found to indicate a spec file")
	}

	// 3. 根据结构特征猜测版本
	_, hasComponents := genericSpec["components"]
	_, hasDefinitions := genericSpec["definitions"]

	// 优先猜测为 OpenAPI v3 (更现代)
	if hasComponents {
		var spec OpenAPISpec
		if err := yaml.Unmarshal(content, &spec); err == nil {
			converter.spec = &spec
			return converter.convertV3()
		}
	}

	// 尝试猜测为 Swagger v2
	if hasDefinitions {
		var spec SwaggerSpec
		if err := yaml.Unmarshal(content, &spec); err == nil {
			converter.spec = &spec
			return converter.convertV2()
		}
	}

	return nil, fmt.Errorf("unsupported or missing 'swagger'/'openapi' version field; fallback attempts to parse as v2 or v3 based on structure also failed")
}

// convertV3 handles the conversion of an OpenAPI 3.0 spec.
func (c *Converter) convertV3() (*idl_ast.IDLSchema, error) {
	spec := c.spec.(*OpenAPISpec)
	if spec.Components != nil {
		c.definitionsMap = spec.Components.Schemas
	}
	c.processComponentsV3(spec.Components)
	if err := c.processPathsV3(spec.Paths); err != nil {
		return nil, err
	}
	c.deduplicateRequestStructs()
	return c.assembleSchema("openapi", spec.OpenAPI), nil
}

// convertV2 handles the conversion of a Swagger 2.0 spec.
func (c *Converter) convertV2() (*idl_ast.IDLSchema, error) {
	spec := c.spec.(*SwaggerSpec)
	c.definitionsMap = spec.Definitions
	c.processDefinitionsV2(spec.Definitions)
	if err := c.processPathsV2(spec.Paths); err != nil {
		return nil, err
	}
	c.deduplicateRequestStructs()
	return c.assembleSchema("thrift", spec.Swagger), nil
}

// resolveSchema follows a $ref string to its definition.
func (c *Converter) resolveSchema(ref string) *Schema {
	if !strings.HasPrefix(ref, "#/definitions/") {
		// Basic support for components, extend as needed
		ref = strings.Replace(ref, "#/components/schemas/", "#/definitions/", 1)
	}

	if strings.HasPrefix(ref, "#/definitions/") {
		defName := strings.TrimPrefix(ref, "#/definitions/")
		if def, ok := c.definitionsMap[defName]; ok {
			return def
		}
	}
	return nil // Or return an error
}

// flattenAllOf merges a list of schemas from an allOf directive into a single schema.
func (c *Converter) flattenAllOf(schemas []*Schema) *Schema {
	finalSchema := &Schema{
		Properties: make(map[string]*Schema),
	}
	requiredSet := make(map[string]bool)

	for _, s := range schemas {
		// Resolve schema if it's a reference
		resolvedSchema := s
		if s.Ref != "" {
			resolved := c.resolveSchema(s.Ref)
			if resolved != nil {
				resolvedSchema = resolved
			}
		}

		// Recursively flatten if the resolved schema also has allOf
		if len(resolvedSchema.AllOf) > 0 {
			resolvedSchema = c.flattenAllOf(resolvedSchema.AllOf)
		}

		// Merge properties
		for key, prop := range resolvedSchema.Properties {
			finalSchema.Properties[key] = prop
		}
		// Merge required fields
		for _, req := range resolvedSchema.Required {
			if !requiredSet[req] {
				finalSchema.Required = append(finalSchema.Required, req)
				requiredSet[req] = true
			}
		}
		if finalSchema.Description == "" && resolvedSchema.Description != "" {
			finalSchema.Description = resolvedSchema.Description
		}

	}
	return finalSchema
}

func (c *Converter) convertSchemaToType(schema *Schema, currentFileNamespace, parentName, fieldName string) *idl_ast.Type {
	if schema == nil {
		return &idl_ast.Type{Name: "void", IsPrimitive: true}
	}

	var ref string
	if schema.Ref != "" {
		ref = schema.Ref
	} else if len(schema.AllOf) == 1 && schema.AllOf[0].Ref != "" {
		ref = schema.AllOf[0].Ref
	}

	if ref != "" {
		resolvedSchema := c.resolveSchema(ref)
		if resolvedSchema != nil && isTypedefCandidate(resolvedSchema) {
			return c.convertSchemaToType(resolvedSchema, currentFileNamespace, parentName, fieldName)
		}

		refFullName := ""
		if strings.HasPrefix(ref, "#/components/schemas/") {
			refFullName = strings.TrimPrefix(ref, "#/components/schemas/")
		} else if strings.HasPrefix(ref, "#/definitions/") {
			refFullName = strings.TrimPrefix(ref, "#/definitions/")
		}

		refNamespace, originalShortName := splitDefinitionName(refFullName)
		shortName := sanitizeAndTransliterateName(originalShortName)
		finalName := ""

		outputDirPrefix := c.getOutputDirPrefix()
		targetFileName := ""
		sanitizedRefNamespace := strings.ReplaceAll(refNamespace, "-", "_") // 在这里提前清理
		if refNamespace == "main" {
			targetFileName = c.getMainThriftFileName()
		} else {
			targetFileName = filepath.Join(outputDirPrefix, sanitizedRefNamespace+".thrift")
		}

		currentFileName := ""
		sanitizedCurrentFileNamespace := strings.ReplaceAll(currentFileNamespace, "-", "_") // 同时清理当前命名空间
		if currentFileNamespace == "main" {
			currentFileName = c.getMainThriftFileName()
		} else {
			currentFileName = filepath.Join(outputDirPrefix, sanitizedCurrentFileNamespace+".thrift")
		}

		if targetFileName == currentFileName {
			finalName = shortName
		} else {
			if refNamespace != "main" {
				finalName = sanitizedRefNamespace + "." + shortName
			} else {
				finalName = shortName
			}
		}

		return &idl_ast.Type{
			Name:               finalName,
			IsPrimitive:        false,
			FullyQualifiedName: fmt.Sprintf("%s#%s", c.filePath, refFullName),
		}
	}

	if len(schema.AllOf) > 0 || (schema.Type == "object" && len(schema.Properties) > 0) {
		baseName := sanitizeAndTransliterateName(toPascalCase(parentName) + toPascalCase(fieldName))
		newStructName := baseName

		outputDirPrefix := c.getOutputDirPrefix()
		targetFileName := ""
		if currentFileNamespace == "main" {
			targetFileName = c.getMainThriftFileName()
		} else {
			targetFileName = filepath.Join(outputDirPrefix, currentFileNamespace+".thrift")
		}
		defs := c.getOrCreateDefs(targetFileName)

		isNameTaken := func(name string) bool {
			for _, msg := range defs.Messages {
				if msg.Name == name {
					return true
				}
			}
			for _, enm := range defs.Enums {
				if enm.Name == name {
					return true
				}
			}
			for _, td := range defs.Typedefs {
				if td.Alias == name {
					return true
				}
			}
			return false
		}

		counter := 2
		for isNameTaken(newStructName) {
			newStructName = fmt.Sprintf("%s_%d", baseName, counter)
			counter++
		}

		newMessage := idl_ast.Message{
			Name:               newStructName,
			FullyQualifiedName: fmt.Sprintf("%s#%s", targetFileName, newStructName),
			Type:               "struct",
			Comments:           descriptionToComments(schema.Description),
		}

		finalSchema := schema
		if len(schema.AllOf) > 0 {
			finalSchema = c.flattenAllOf(schema.AllOf)
		}
		if len(schema.Properties) > 0 {
			if finalSchema.Properties == nil {
				finalSchema.Properties = make(map[string]*Schema)
			}
			for k, v := range schema.Properties {
				finalSchema.Properties[k] = v
			}
		}

		requiredMap := make(map[string]bool)
		for _, reqField := range finalSchema.Required {
			requiredMap[reqField] = true
		}

		propNames := make([]string, 0, len(finalSchema.Properties))
		for propName := range finalSchema.Properties {
			propNames = append(propNames, propName)
		}
		sort.Strings(propNames)

		for i, propName := range propNames {
			propSchema := finalSchema.Properties[propName]
			required := "optional"
			if requiredMap[propName] {
				required = "required"
			}
			field := idl_ast.Field{
				ID:       i + 1,
				Name:     propName,
				Type:     *c.convertSchemaToType(propSchema, currentFileNamespace, newStructName, propName),
				Required: required,
				Comments: descriptionToComments(propSchema.Description),
			}
			newMessage.Fields = append(newMessage.Fields, field)
		}

		defs.Messages = append(defs.Messages, newMessage)
		return &idl_ast.Type{Name: newStructName, IsPrimitive: false}
	}

	switch schema.Type {
	case "string":
		return &idl_ast.Type{Name: "string", IsPrimitive: true}
	case "integer":
		if schema.Format == "int64" {
			return &idl_ast.Type{Name: "i64", IsPrimitive: true}
		}
		return &idl_ast.Type{Name: "i32", IsPrimitive: true}
	case "number":
		if schema.Format == "float" {
			return &idl_ast.Type{Name: "float", IsPrimitive: true}
		}
		return &idl_ast.Type{Name: "double", IsPrimitive: true}
	case "boolean":
		return &idl_ast.Type{Name: "bool", IsPrimitive: true}
	case "array":
		itemName := strings.TrimSuffix(fieldName, "s")
		if itemName == fieldName {
			itemName = fieldName + "Item"
		}
		return &idl_ast.Type{
			Name:      "list",
			ValueType: c.convertSchemaToType(schema.Items, currentFileNamespace, parentName, itemName),
		}
	case "object", "":
		if schema.AdditionalProperties != nil {
			var apSchema *Schema
			switch ap := schema.AdditionalProperties.(type) {
			case bool:
				if ap {
					return &idl_ast.Type{
						Name:      "map",
						KeyType:   &idl_ast.Type{Name: "string", IsPrimitive: true},
						ValueType: &idl_ast.Type{Name: "string", IsPrimitive: true},
					}
				}
			case map[string]any:
				tempSchema := &Schema{}
				yamlBytes, err := yaml.Marshal(ap)
				if err == nil {
					if yaml.Unmarshal(yamlBytes, tempSchema) == nil {
						apSchema = tempSchema
					}
				}
			}
			if apSchema != nil {
				isAnyType := apSchema.Type == "" && apSchema.Ref == "" && len(apSchema.Properties) == 0 && apSchema.AdditionalProperties == nil
				valueType := &idl_ast.Type{Name: "string", IsPrimitive: true}
				if !isAnyType {
					valueType = c.convertSchemaToType(apSchema, currentFileNamespace, parentName, fieldName+"Value")
				}
				return &idl_ast.Type{
					Name:      "map",
					KeyType:   &idl_ast.Type{Name: "string", IsPrimitive: true},
					ValueType: valueType,
				}
			}
		}
		return &idl_ast.Type{
			Name:      "map",
			KeyType:   &idl_ast.Type{Name: "string", IsPrimitive: true},
			ValueType: &idl_ast.Type{Name: "string", IsPrimitive: true},
		}
	default:
		return &idl_ast.Type{Name: "string", IsPrimitive: true}
	}
}

func (c *Converter) convertValueToConstantValue(val any) (*idl_ast.ConstantValue, error) {
	if val == nil {
		return nil, nil
	}
	cv := &idl_ast.ConstantValue{}
	switch v := val.(type) {
	case string:
		cv.Value = strconv.Quote(v)
	case int:
		cv.Value = int64(v)
	case int64, float64, bool:
		cv.Value = v
	case []any:
		var list []*idl_ast.ConstantValue
		for _, item := range v {
			itemCV, err := c.convertValueToConstantValue(item)
			if err != nil {
				return nil, err
			}
			list = append(list, itemCV)
		}
		cv.Value = list
	case map[string]any:
		var entries []*idl_ast.ConstantMapEntry
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := v[key]
			keyCV := &idl_ast.ConstantValue{Value: strconv.Quote(key)}
			valueCV, err := c.convertValueToConstantValue(value)
			if err != nil {
				return nil, err
			}
			entries = append(entries, &idl_ast.ConstantMapEntry{Key: keyCV, Value: valueCV})
		}
		cv.Value = entries
	default:
		return nil, fmt.Errorf("unsupported default value type: %T", val)
	}
	return cv, nil
}
