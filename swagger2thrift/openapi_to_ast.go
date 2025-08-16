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

	} else if openAPIVersion, ok := genericSpec["openapi"].(string); ok && strings.HasPrefix(openAPIVersion, "3.") {
		var spec OpenAPISpec
		if err := yaml.Unmarshal(content, &spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OpenAPI v3 spec: %w", err)
		}
		converter.spec = &spec
		return converter.convertV3()
	}

	return nil, fmt.Errorf("unsupported or missing 'swagger'/'openapi' version field")
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

func (c *Converter) convertSchemaToType(schema *Schema, parentNamespace, parentName, fieldName string) *idl_ast.Type {
	if schema == nil {
		return &idl_ast.Type{Name: "void", IsPrimitive: true}
	}

	// 1. 最高优先级：处理引用。如果 schema 是一个引用，直接返回引用名，不再进行任何内部解析。
	//    这是为了确保全局定义（如 enum, struct）只由 processSchemas 处理一次。
	if schema.Ref != "" {
		refOriginalFullName := ""
		if strings.HasPrefix(schema.Ref, "#/components/schemas/") {
			refOriginalFullName = strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		} else if strings.HasPrefix(schema.Ref, "#/definitions/") {
			refOriginalFullName = strings.TrimPrefix(schema.Ref, "#/definitions/")
		} else {
			// Fallback for simple references like "$ref": "MyType"
			refOriginalFullName = schema.Ref
		}

		refNamespace, refOriginalShortName := splitDefinitionName(refOriginalFullName)
		sanitizedShortName := sanitizeAndTransliterateName(refOriginalShortName)

		var finalName string
		if refNamespace != "main" {
			finalName = refNamespace + "." + sanitizedShortName
		} else {
			finalName = sanitizedShortName
		}

		return &idl_ast.Type{
			Name:               finalName,
			IsPrimitive:        false,
			FullyQualifiedName: fmt.Sprintf("%s#%s", c.filePath, refOriginalFullName),
		}
	}

	// 2. 处理 allOf，这可能会将多个 schema (包括 $ref) 合并。
	if len(schema.AllOf) > 0 {
		// 特殊情况：如果 allOf 只有一个 $ref 元素，直接处理该 $ref
		if len(schema.AllOf) == 1 && schema.AllOf[0].Ref != "" {
			return c.convertSchemaToType(schema.AllOf[0], parentNamespace, parentName, fieldName)
		}
		// 否则，压平 allOf
		schema = c.flattenAllOf(schema.AllOf)
	}

	// 3. 处理内联（匿名）的整型枚举。这段逻辑现在只会在 schema 没有 $ref 时执行。
	if len(schema.Enum) > 0 && schema.Type == "integer" {
		if parentName != "" && fieldName != "" {
			enumName := sanitizeAndTransliterateName(toPascalCase(parentName) + toPascalCase(fieldName))
			var targetFileName string
			outputDir := c.getOutputDirPrefix()
			if parentNamespace == "main" {
				targetFileName = c.getMainThriftFileName()
			} else {
				targetFileName = filepath.Join(outputDir, parentNamespace+".thrift")
			}
			defs := c.getOrCreateDefs(targetFileName)

			enumExists := false
			for _, e := range defs.Enums {
				if e.Name == enumName {
					enumExists = true
					break
				}
			}

			if !enumExists {
				newEnum := idl_ast.Enum{
					Name:               enumName,
					FullyQualifiedName: fmt.Sprintf("%s#%s", targetFileName, enumName),
					Comments:           descriptionToComments(schema.Description),
				}
				useVarNames := len(schema.XEnumVarNames) == len(schema.Enum)
				for i, val := range schema.Enum {
					var memberName string
					if useVarNames {
						memberName = schema.XEnumVarNames[i]
					} else {
						memberName = fmt.Sprintf("%s_%v", enumName, val)
					}
					intValue, ok := val.(int)
					if !ok {
						if floatVal, isFloat := val.(float64); isFloat {
							intValue = int(floatVal)
						} else {
							intValue = i
						}
					}
					newEnum.Values = append(newEnum.Values, idl_ast.EnumValue{
						Name:  memberName,
						Value: intValue,
					})
				}
				defs.Enums = append(defs.Enums, newEnum)
			}
			return &idl_ast.Type{Name: enumName, IsPrimitive: false}
		}
	}

	// 4. 处理基本类型、map 和内联 struct
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
			ValueType: c.convertSchemaToType(schema.Items, parentNamespace, parentName, itemName),
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
				if isAnyType {
					return &idl_ast.Type{
						Name:      "map",
						KeyType:   &idl_ast.Type{Name: "string", IsPrimitive: true},
						ValueType: &idl_ast.Type{Name: "string", IsPrimitive: true},
					}
				}
				return &idl_ast.Type{
					Name:      "map",
					KeyType:   &idl_ast.Type{Name: "string", IsPrimitive: true},
					ValueType: c.convertSchemaToType(apSchema, parentNamespace, parentName, fieldName+"Value"),
				}
			}
		}
		if len(schema.Properties) > 0 {
			newStructName := sanitizeAndTransliterateName(toPascalCase(parentName) + toPascalCase(fieldName))
			var targetFileName string
			outputDir := c.getOutputDirPrefix()
			if parentNamespace == "main" {
				targetFileName = c.getMainThriftFileName()
			} else {
				targetFileName = filepath.Join(outputDir, parentNamespace+".thrift")
			}
			defs := c.getOrCreateDefs(targetFileName)

			for _, msg := range defs.Messages {
				if msg.Name == newStructName {
					return &idl_ast.Type{Name: newStructName, IsPrimitive: false}
				}
			}

			newMessage := idl_ast.Message{
				Name:               newStructName,
				FullyQualifiedName: fmt.Sprintf("%s#%s", targetFileName, newStructName),
				Type:               "struct",
				Comments:           descriptionToComments(schema.Description),
			}

			requiredMap := make(map[string]bool)
			for _, reqField := range schema.Required {
				requiredMap[reqField] = true
			}

			propNames := make([]string, 0, len(schema.Properties))
			for propName := range schema.Properties {
				propNames = append(propNames, propName)
			}
			sort.Strings(propNames)

			for i, propName := range propNames {
				propSchema := schema.Properties[propName]
				required := "optional"
				if requiredMap[propName] {
					required = "required"
				}
				field := idl_ast.Field{
					ID:       i + 1,
					Name:     propName,
					Type:     *c.convertSchemaToType(propSchema, parentNamespace, newStructName, propName),
					Required: required,
					Comments: descriptionToComments(propSchema.Description),
				}
				newMessage.Fields = append(newMessage.Fields, field)
			}
			defs.Messages = append(defs.Messages, newMessage)
			return &idl_ast.Type{Name: newStructName, IsPrimitive: false}
		}

		return &idl_ast.Type{Name: "map", KeyType: &idl_ast.Type{Name: "string", IsPrimitive: true}, ValueType: &idl_ast.Type{Name: "string", IsPrimitive: true}}
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
