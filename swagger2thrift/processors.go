package swagger2thrift

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Skyenought/idlanalyzer/idl_ast"
)

func (c *Converter) processComponentsV3(components *Components) {
	if components != nil {
		c.processSchemas(components.Schemas)
	}
}

func (c *Converter) processDefinitionsV2(definitions map[string]*Schema) {
	c.processSchemas(definitions)
}

func isTypedefCandidate(schema *Schema) bool {
	if len(schema.Properties) > 0 {
		return false
	}
	switch schema.Type {
	case "string", "integer", "number", "boolean", "array":
		return true
	case "object", "":
		return schema.AdditionalProperties != nil
	default:
		return false
	}
}

func (c *Converter) processSchemas(schemas map[string]*Schema) {
	if schemas == nil {
		return
	}

	schemaNames := make([]string, 0, len(schemas))
	for name := range schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	for _, name := range schemaNames {
		schema := schemas[name]

		namespace, shortName := splitDefinitionName(name)

		var fileName string
		outputDir := c.getOutputDirPrefix() // 获取共同的输出目录, e.g., "docs_swagger"

		if namespace == "main" {
			fileName = c.getMainThriftFileName() // "docs_swagger/docs_swagger.thrift"
		} else {
			fileName = filepath.Join(outputDir, namespace+".thrift") // "docs_swagger/payload.thrift"
		}

		defs := c.getOrCreateDefs(fileName)
		fqn := fmt.Sprintf("%s#%s", fileName, shortName)

		if len(schema.Enum) > 0 {
			enum := idl_ast.Enum{
				Name:               shortName,
				FullyQualifiedName: fqn,
				Comments:           descriptionToComments(schema.Description),
			}

			// 检查 x-enum-varnames 是否存在且与 enum 列表长度匹配
			useVarNames := len(schema.XEnumVarNames) == len(schema.Enum)

			for i, val := range schema.Enum {
				var memberName string
				if useVarNames {
					// 正确做法：使用 x-enum-varnames 中提供的名称
					memberName = schema.XEnumVarNames[i]
				} else {
					memberName = fmt.Sprintf("%s_%v", toPascalCase(shortName), val)
				}

				// 将 enum 的值转换为整数
				intValue, ok := val.(int)
				if !ok {
					if floatVal, isFloat := val.(float64); isFloat {
						intValue = int(floatVal)
					} else {
						intValue = i
					}
				}

				enum.Values = append(enum.Values, idl_ast.EnumValue{
					Name:  memberName,
					Value: intValue,
				})
			}
			defs.Enums = append(defs.Enums, enum)
		} else if isTypedefCandidate(schema) {
			typedef := idl_ast.Typedef{
				Alias:    shortName,
				Type:     *c.convertSchemaToType(schema, namespace, "", ""),
				Comments: descriptionToComments(schema.Description),
			}
			defs.Typedefs = append(defs.Typedefs, typedef)
		} else {
			message := idl_ast.Message{
				Name:               shortName,
				FullyQualifiedName: fqn,
				Type:               "struct",
				Comments:           descriptionToComments(schema.Description),
			}
			requiredMap := make(map[string]bool)
			for _, fieldName := range schema.Required {
				requiredMap[fieldName] = true
			}
			propNames := make([]string, 0, len(schema.Properties))
			for propName := range schema.Properties {
				propNames = append(propNames, propName)
			}
			sort.Strings(propNames)
			fieldID := 1
			for _, propName := range propNames {
				propSchema := schema.Properties[propName]
				required := "optional"
				if requiredMap[propName] {
					required = "required"
				}
				defaultValue, _ := c.convertValueToConstantValue(propSchema.Default)
				field := idl_ast.Field{
					ID:           fieldID,
					Name:         propName,
					Type:         *c.convertSchemaToType(propSchema, namespace, shortName, propName),
					Required:     required,
					DefaultValue: defaultValue,
					Comments:     descriptionToComments(propSchema.Description),
				}
				message.Fields = append(message.Fields, field)
				fieldID++
			}
			defs.Messages = append(defs.Messages, message)
		}
	}
}

func (c *Converter) processPathsV3(paths map[string]*PathItem) error {
	pathKeys := make([]string, 0, len(paths))
	for k := range paths {
		pathKeys = append(pathKeys, k)
	}
	sort.Strings(pathKeys)

	for _, path := range pathKeys {
		pathItem := paths[path]
		operations := map[string]*Operation{
			"get": pathItem.Get, "put": pathItem.Put, "post": pathItem.Post,
			"delete": pathItem.Delete, "patch": pathItem.Patch,
		}

		// 定义一个固定的 HTTP 方法处理顺序，以保证确定性
		httpMethods := []string{"get", "put", "post", "delete", "patch"}

		// 按照固定顺序遍历 HTTP 方法
		for _, httpMethod := range httpMethods {
			op := operations[httpMethod]
			if op == nil {
				continue
			}

			_, baseFuncName, _ := c.getServiceAndFuncNames(op.Tags, op.OperationID, httpMethod, path, "")
			// 强制生成 Response 类型，即使没有 schema
			returnType := c.findBestReturnTypeV3(op.Responses, baseFuncName)
			service, baseFuncName, reqName := c.getServiceAndFuncNames(op.Tags, op.OperationID, httpMethod, path, returnType.Name)

			defs := c.getOrCreateDefs(c.getMainThriftFileName())
			var servicePtr *idl_ast.Service
			for i := range defs.Services {
				if defs.Services[i].Name == service.Name {
					servicePtr = &defs.Services[i]
					break
				}
			}
			if servicePtr == nil {
				defs.Services = append(defs.Services, service)
				servicePtr = &defs.Services[len(defs.Services)-1]
			}

			funcName := c.disambiguateFunctionName(baseFuncName, path, servicePtr)

			params, throws := c.processParamsAndBodyV3(op.Parameters, op.RequestBody, op.Responses)
			function := idl_ast.Function{
				Name:               funcName,
				FullyQualifiedName: fmt.Sprintf("%s#%s.%s", c.getMainThriftFileName(), service.Name, funcName),
				ReturnType:         returnType,
				Throws:             throws,
				Comments:           descriptionToComments(op.Description),
				Annotations: []idl_ast.Annotation{{
					Name:  fmt.Sprintf("api.%s", httpMethod),
					Value: &idl_ast.ConstantValue{Value: fmt.Sprintf(`"%s"`, formatPathForAnnotation(path))},
				}},
			}

			// 无论参数是否为空，都创建 Request 结构体
			reqStruct := idl_ast.Message{
				Name:               reqName,
				Type:               "struct",
				FullyQualifiedName: fmt.Sprintf("%s#%s", c.getMainThriftFileName(), reqName),
				Fields:             params, // 如果 params 为空，这里就是空 struct
			}
			c.requestStructs[reqName] = append(c.requestStructs[reqName], reqStruct)

			// 并且总是为函数添加参数
			function.Parameters = []idl_ast.Field{
				{ID: 1, Name: "request", Type: idl_ast.Type{Name: reqName}},
			}

			servicePtr.Functions = append(servicePtr.Functions, function)
		}
	}
	return nil
}

func (c *Converter) processPathsV2(paths map[string]*SwaggerPathItem) error {
	pathKeys := make([]string, 0, len(paths))
	for k := range paths {
		pathKeys = append(pathKeys, k)
	}
	sort.Strings(pathKeys)

	for _, path := range pathKeys {
		pathItem := paths[path]
		operations := map[string]*SwaggerOperation{
			"get": pathItem.Get, "put": pathItem.Put, "post": pathItem.Post,
			"delete": pathItem.Delete, "patch": pathItem.Patch,
		}

		// 定义一个固定的 HTTP 方法处理顺序，以保证确定性
		httpMethods := []string{"get", "put", "post", "delete", "patch"}

		// 按照固定顺序遍历 HTTP 方法
		for _, httpMethod := range httpMethods {
			op := operations[httpMethod]
			if op == nil {
				continue
			}

			_, baseFuncName, _ := c.getServiceAndFuncNames(op.Tags, op.OperationID, httpMethod, path, "")
			// 强制生成 Response 类型，即使没有 schema
			returnType := c.findBestReturnTypeV2(op.Responses, baseFuncName)
			service, baseFuncName, reqName := c.getServiceAndFuncNames(op.Tags, op.OperationID, httpMethod, path, returnType.Name)

			defs := c.getOrCreateDefs(c.getMainThriftFileName())
			var servicePtr *idl_ast.Service
			for i := range defs.Services {
				if defs.Services[i].Name == service.Name {
					servicePtr = &defs.Services[i]
					break
				}
			}
			if servicePtr == nil {
				defs.Services = append(defs.Services, service)
				servicePtr = &defs.Services[len(defs.Services)-1]
			}

			funcName := c.disambiguateFunctionName(baseFuncName, path, servicePtr)
			params, throws := c.processParamsV2(op.Parameters, op.Responses)

			function := idl_ast.Function{
				Name:               funcName,
				FullyQualifiedName: fmt.Sprintf("%s#%s.%s", c.getMainThriftFileName(), service.Name, funcName),
				ReturnType:         returnType,
				Throws:             throws,
				Comments:           descriptionToComments(op.Description),
				Annotations: []idl_ast.Annotation{{
					Name:  fmt.Sprintf("api.%s", httpMethod),
					Value: &idl_ast.ConstantValue{Value: fmt.Sprintf(`"%s"`, formatPathForAnnotation(path))},
				}},
			}

			// 无论参数是否为空，都创建 Request 结构体
			reqStruct := idl_ast.Message{
				Name:               reqName,
				Type:               "struct",
				FullyQualifiedName: fmt.Sprintf("%s#%s", c.getMainThriftFileName(), reqName),
				Fields:             params, // 如果 params 为空，这里就是空 struct
			}
			c.requestStructs[reqName] = append(c.requestStructs[reqName], reqStruct)

			// 并且总是为函数添加参数
			function.Parameters = []idl_ast.Field{
				{ID: 1, Name: "request", Type: idl_ast.Type{Name: reqName}},
			}

			servicePtr.Functions = append(servicePtr.Functions, function)
		}
	}
	return nil
}
func (c *Converter) processParamsAndBodyV3(params []*Parameter, reqBody *RequestBody, responses map[string]*Response) ([]idl_ast.Field, []idl_ast.Field) {
	var astThrows []idl_ast.Field
	for code, resp := range responses {
		if !strings.HasPrefix(code, "2") {
			if resp.Content != nil {
				if mediaType, ok := resp.Content["application/json"]; ok {
					astThrows = append(astThrows, idl_ast.Field{
						ID:       len(astThrows) + 1,
						Name:     "error" + code,
						Type:     *c.convertSchemaToType(mediaType.Schema, "main", "error"+code, ""),
						Comments: descriptionToComments(resp.Description),
					})
				}
			}
		}
	}

	paramFields := make(map[string]*idl_ast.Field)
	currentNamespace := "main"

	for _, param := range params {
		if existingField, ok := paramFields[param.Name]; ok {
			if annotation := createParameterAnnotation(param.In, param.Name); annotation != nil {
				existingField.Annotations = append(existingField.Annotations, *annotation)
			}
			if param.Required {
				existingField.Required = "required"
			}
		} else {
			required := "optional"
			if param.Required {
				required = "required"
			}
			field := &idl_ast.Field{
				Name:     param.Name,
				Type:     *c.convertSchemaToType(param.Schema, currentNamespace, "", param.Name),
				Required: required,
				Comments: descriptionToComments(param.Description),
			}
			if annotation := createParameterAnnotation(param.In, param.Name); annotation != nil {
				field.Annotations = append(field.Annotations, *annotation)
			}
			paramFields[param.Name] = field
		}
	}

	if reqBody != nil && reqBody.Content != nil {
		if mediaType, ok := reqBody.Content["application/json"]; ok && mediaType.Schema != nil {
			if mediaType.Schema.Ref == "" && (mediaType.Schema.Type == "object" || mediaType.Schema.Type == "") {
				for propName, propSchema := range mediaType.Schema.Properties {
					required := "optional"
					// This is a simplified check for required. A full implementation would check reqBody.Schema.Required array.
					field := &idl_ast.Field{
						Name:     propName,
						Type:     *c.convertSchemaToType(propSchema, currentNamespace, "", propName),
						Required: required,
						Comments: descriptionToComments(propSchema.Description),
					}
					if annotation := createParameterAnnotation("body", propName); annotation != nil {
						field.Annotations = append(field.Annotations, *annotation)
					}
					paramFields[propName] = field
				}
			} else {
				required := "optional"
				if reqBody.Required {
					required = "required"
				}
				field := &idl_ast.Field{
					Name:     "body",
					Type:     *c.convertSchemaToType(mediaType.Schema, currentNamespace, "body", ""),
					Required: required,
					Comments: descriptionToComments(reqBody.Description),
				}
				if annotation := createParameterAnnotation("body", "body"); annotation != nil {
					field.Annotations = append(field.Annotations, *annotation)
				}
				paramFields["body"] = field
			}
		}
	}

	var astParams []idl_ast.Field
	paramNames := make([]string, 0, len(paramFields))
	for name := range paramFields {
		paramNames = append(paramNames, name)
	}
	sort.Strings(paramNames)

	for i, name := range paramNames {
		field := paramFields[name]
		field.ID = i + 1
		astParams = append(astParams, *field)
	}

	return astParams, astThrows
}

func (c *Converter) processParamsV2(params []*SwaggerParameter, responses map[string]*SwaggerResponse) ([]idl_ast.Field, []idl_ast.Field) {
	var astThrows []idl_ast.Field
	for code, resp := range responses {
		if !strings.HasPrefix(code, "2") {
			if resp.Schema != nil {
				astThrows = append(astThrows, idl_ast.Field{
					ID:       len(astThrows) + 1,
					Name:     "error" + code,
					Type:     *c.convertSchemaToType(resp.Schema, "main", "error"+code, ""),
					Comments: descriptionToComments(resp.Description),
				})
			}
		}
	}

	paramFields := make(map[string]*idl_ast.Field)
	currentNamespace := "main"

	for _, param := range params {
		if param.In == "body" && param.Schema != nil && param.Schema.Ref == "" && (param.Schema.Type == "object" || param.Schema.Type == "") {
			requiredMap := make(map[string]bool)
			for _, reqField := range param.Schema.Required {
				requiredMap[reqField] = true
			}

			for propName, propSchema := range param.Schema.Properties {
				required := "optional"
				if requiredMap[propName] {
					required = "required"
				}
				field := &idl_ast.Field{
					Name:     propName,
					Type:     *c.convertSchemaToType(propSchema, currentNamespace, param.Name, propName),
					Required: required,
					Comments: descriptionToComments(propSchema.Description),
				}
				if annotation := createParameterAnnotation("body", propName); annotation != nil {
					field.Annotations = append(field.Annotations, *annotation)
				}
				paramFields[propName] = field
			}
			continue
		}

		// MODIFICATION: Sanitize field name
		sanitizedName := toLowerCamelCase(param.Name)

		if existingField, ok := paramFields[sanitizedName]; ok {
			if annotation := createParameterAnnotation(param.In, param.Name); annotation != nil {
				existingField.Annotations = append(existingField.Annotations, *annotation)
			}
			if param.Required {
				existingField.Required = "required"
			}
		} else {
			required := "optional"
			if param.Required {
				required = "required"
			}
			var paramType idl_ast.Type
			if param.In == "body" {
				paramType = *c.convertSchemaToType(param.Schema, currentNamespace, param.Name, "")
			} else {
				paramTypeName := param.Type
				if paramTypeName == "" && param.Schema == nil {
					paramTypeName = "string"
				}
				paramType = *c.convertSchemaToType(&Schema{Type: paramTypeName, Format: param.Format, Items: param.Items}, currentNamespace, "", param.Name)
			}
			field := &idl_ast.Field{
				Name:     sanitizedName, // Use sanitized name
				Type:     paramType,
				Required: required,
				Comments: descriptionToComments(param.Description),
			}
			// Use original name for the annotation's value
			if annotation := createParameterAnnotation(param.In, param.Name); annotation != nil {
				field.Annotations = append(field.Annotations, *annotation)
			}
			paramFields[sanitizedName] = field
		}
	}

	var astParams []idl_ast.Field
	paramNames := make([]string, 0, len(paramFields))
	for name := range paramFields {
		paramNames = append(paramNames, name)
	}
	sort.Strings(paramNames)

	for i, name := range paramNames {
		field := paramFields[name]
		field.ID = i + 1
		astParams = append(astParams, *field)
	}

	return astParams, astThrows
}

func (c *Converter) findBestReturnTypeV3(responses map[string]*Response, baseFuncName string) idl_ast.Type {
	if responses == nil {
		return c.createEmptyResponseStruct(baseFuncName)
	}

	var bestSchema *Schema
	if resp, ok := responses["200"]; ok {
		if resp.Content != nil {
			if mediaType, ok := resp.Content["application/json"]; ok {
				bestSchema = mediaType.Schema
			}
		}
	}

	if bestSchema == nil {
		var sortedCodes []string
		for code := range responses {
			sortedCodes = append(sortedCodes, code)
		}
		sort.Strings(sortedCodes)

		for _, code := range sortedCodes {
			if strings.HasPrefix(code, "2") {
				if resp := responses[code]; resp.Content != nil {
					if mediaType, ok := resp.Content["application/json"]; ok {
						bestSchema = mediaType.Schema
						break
					}
				}
			}
		}
	}

	if bestSchema == nil {
		return c.createEmptyResponseStruct(baseFuncName)
	}

	return *c.convertSchemaToType(bestSchema, "main", baseFuncName, "Response")
}

func (c *Converter) findBestReturnTypeV2(responses map[string]*SwaggerResponse, baseFuncName string) idl_ast.Type {
	if responses == nil {
		// 修改点：不再返回 void，而是创建空的 Response struct
		return c.createEmptyResponseStruct(baseFuncName)
	}

	var bestResp *SwaggerResponse
	if resp, ok := responses["200"]; ok && resp.Schema != nil {
		bestResp = resp
	} else {
		var sortedCodes []string
		for code := range responses {
			sortedCodes = append(sortedCodes, code)
		}
		sort.Strings(sortedCodes)

		for _, code := range sortedCodes {
			if strings.HasPrefix(code, "2") {
				if resp, ok := responses[code]; ok {
					bestResp = resp
					break
				}
			}
		}
	}

	if bestResp == nil || bestResp.Schema == nil {
		return c.createEmptyResponseStruct(baseFuncName)
	}

	return *c.convertSchemaToType(bestResp.Schema, "main", baseFuncName, "Response")
}

func (c *Converter) getServiceAndFuncNames(tags []string, opID, httpMethod, path, responseTypeName string) (idl_ast.Service, string, string) {
	serviceName := "HTTPService" // Default name
	if c.cfg != nil && c.cfg.ServiceName != "" {
		serviceName = c.cfg.ServiceName
	}

	service := idl_ast.Service{
		Name:               serviceName,
		FullyQualifiedName: fmt.Sprintf("%s#%s", c.getMainThriftFileName(), serviceName),
	}

	funcName := getFunctionName(opID, httpMethod, path, responseTypeName)
	reqName := funcName + "Request"

	return service, funcName, reqName
}

func (c *Converter) disambiguateFunctionName(baseFuncName, path string, service *idl_ast.Service) string {
	isTaken := func(name string, functions []idl_ast.Function) bool {
		for _, f := range functions {
			if f.Name == name {
				return true
			}
		}
		return false
	}

	finalName := baseFuncName
	if isTaken(finalName, service.Functions) {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) > 0 {
			for i := len(parts) - 1; i >= 0; i-- {
				if !strings.HasPrefix(parts[i], "{") {
					finalName += "For" + toPascalCase(parts[i])
					break
				}
			}
		}
	}

	counter := 2
	originalName := finalName
	for isTaken(finalName, service.Functions) {
		finalName = fmt.Sprintf("%s%d", originalName, counter)
		counter++
	}

	return finalName
}

func (c *Converter) deduplicateRequestStructs() {
	mainDefs := c.getOrCreateDefs(c.getMainThriftFileName())
	finalRequestStructs := make(map[string]idl_ast.Message)

	for name, structs := range c.requestStructs {
		if len(structs) == 1 {
			finalRequestStructs[name] = structs[0]
			continue
		}

		supersetFieldsMap := make(map[string]idl_ast.Field)
		for _, s := range structs {
			for _, f := range s.Fields {
				if existingField, ok := supersetFieldsMap[f.Name]; ok {
					annotationSet := make(map[string]struct{})
					for _, ann := range existingField.Annotations {
						annotationSet[ann.Name] = struct{}{}
					}

					for _, newAnn := range f.Annotations {
						if _, exists := annotationSet[newAnn.Name]; !exists {
							existingField.Annotations = append(existingField.Annotations, newAnn)
							annotationSet[newAnn.Name] = struct{}{}
						}
					}
					supersetFieldsMap[f.Name] = existingField
				} else {
					supersetFieldsMap[f.Name] = f
				}
			}
		}

		var mergedFields []idl_ast.Field
		for _, field := range supersetFieldsMap {
			mergedFields = append(mergedFields, field)
		}

		sort.Slice(mergedFields, func(i, j int) bool {
			return mergedFields[i].Name < mergedFields[j].Name
		})

		superset := structs[0]
		superset.Fields = mergedFields
		finalRequestStructs[name] = superset
	}

	sortedNames := make([]string, 0, len(finalRequestStructs))
	for name := range finalRequestStructs {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		structToAdd := finalRequestStructs[name]
		for i := range structToAdd.Fields {
			structToAdd.Fields[i].ID = i + 1
		}
		mainDefs.Messages = append(mainDefs.Messages, structToAdd)
	}
}

func getFunctionName(opID, method, path, responseTypeName string) string {
	var funcName string
	if opID != "" {
		funcName = toPascalCase(method) + toPascalCase(opID)
	} else {
		var baseName string
		if responseTypeName != "void" && responseTypeName != "binary" && responseTypeName != "" && !strings.HasPrefix(responseTypeName, "map<") {
			if strings.HasPrefix(responseTypeName, "list<") {
				innerType := strings.TrimSuffix(strings.TrimPrefix(responseTypeName, "list<"), ">")
				_, shortName := splitDefinitionName(innerType)
				baseName = "List" + toPascalCase(shortName) + "s"
			} else {
				_, shortName := splitDefinitionName(responseTypeName)
				baseName = strings.TrimSuffix(shortName, "Response")
			}
		} else {
			path = strings.ReplaceAll(path, "/", " ")
			path = formatPathForAnnotation(path)
			path = strings.ReplaceAll(path, ":", " by ")
			baseName = path
		}
		funcName = toPascalCase(method) + toPascalCase(baseName)
	}
	// Sanitize the name to remove repeated words.
	return sanitizeName(funcName)
}

func (c *Converter) createEmptyResponseStruct(baseFuncName string) idl_ast.Type {
	respName := baseFuncName + "EmptyResponse"
	defs := c.getOrCreateDefs(c.getMainThriftFileName())

	for _, msg := range defs.Messages {
		if msg.Name == respName {
			return idl_ast.Type{Name: respName, IsPrimitive: false}
		}
	}

	emptyStruct := idl_ast.Message{
		Name:               respName,
		Type:               "struct",
		FullyQualifiedName: fmt.Sprintf("%s#%s", c.getMainThriftFileName(), respName),
		Fields:             []idl_ast.Field{},
	}
	defs.Messages = append(defs.Messages, emptyStruct)

	return idl_ast.Type{Name: respName, IsPrimitive: false}
}
