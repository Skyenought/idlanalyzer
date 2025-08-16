package swagger2thrift

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/Skyenought/idlanalyzer/idl_ast"
	"github.com/mozillazg/go-pinyin"
)

// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
	var result strings.Builder
	capitalizeNext := true
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// sanitizeAndTransliterateName converts a string into a valid Thrift identifier.
// It handles Chinese characters by converting them to Pinyin, removes invalid symbols,
// and de-duplicates repeated words in the resulting PascalCase name.
func sanitizeAndTransliterateName(name string) string {
	// Use a regex to find contiguous blocks of Chinese characters.
	hanRegex := regexp.MustCompile(`[\p{Han}]+`)
	pArgs := pinyin.NewArgs()

	// Replace each block of Chinese characters with its PascalCase Pinyin representation.
	processedName := hanRegex.ReplaceAllStringFunc(name, func(s string) string {
		pinyinSlice := pinyin.LazyPinyin(s, pArgs)
		var pinyinBuilder strings.Builder
		for _, p := range pinyinSlice {
			// Capitalize the first letter of each pinyin syllable.
			if len(p) > 0 {
				pinyinBuilder.WriteString(strings.ToUpper(p[:1]) + p[1:])
			}
		}
		return pinyinBuilder.String()
	})

	// Sanitize the result to create a valid PascalCase identifier.
	var result strings.Builder
	capitalizeNext := true
	for _, r := range processedName {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}
	finalName := result.String()

	// De-duplicate consecutively repeated words (e.g., "GetUserUser" becomes "GetUser").
	dedupRegex := regexp.MustCompile(`([A-Z]+[a-z0-9]*)`)
	parts := dedupRegex.FindAllString(finalName, -1)
	if len(parts) <= 1 {
		return finalName
	}

	var cleanParts []string
	if len(parts) > 0 {
		cleanParts = append(cleanParts, parts[0])
	}

	for i := 1; i < len(parts); i++ {
		if parts[i] != parts[i-1] {
			cleanParts = append(cleanParts, parts[i])
		}
	}

	return strings.Join(cleanParts, "")
}

// descriptionToComments converts a description string into an AST Comment block.
func descriptionToComments(desc string) []idl_ast.Comment {
	trimmedDesc := strings.TrimSpace(desc)
	if trimmedDesc == "" {
		return nil
	}

	lines := strings.Split(trimmedDesc, "\n")
	var formattedComment strings.Builder

	if len(lines) == 1 {
		formattedComment.WriteString(fmt.Sprintf("/** %s */", lines[0]))
	} else {
		formattedComment.WriteString("/**\n")
		for _, line := range lines {
			formattedComment.WriteString(fmt.Sprintf(" * %s\n", strings.TrimSpace(line)))
		}
		formattedComment.WriteString(" */")
	}

	return []idl_ast.Comment{{Text: formattedComment.String()}}
}

var pathParamRegex = regexp.MustCompile(`{([^{}]+)}`)

// formatPathForAnnotation converts OpenAPI path templates {param} to Thrift annotation style :param.
func formatPathForAnnotation(path string) string {
	return pathParamRegex.ReplaceAllString(path, ":$1")
}

// splitDefinitionName splits a full definition name like "namespace.TypeName" into its parts.
func splitDefinitionName(fullName string) (namespace, shortName string) {
	lastDot := strings.LastIndex(fullName, ".")
	if lastDot == -1 {
		return "main", fullName
	}
	return fullName[:lastDot], fullName[lastDot+1:]
}

// createParameterAnnotation creates an AST Annotation for a given parameter location and name.
func createParameterAnnotation(in, name string) *idl_ast.Annotation {
	var annotationName string
	switch in {
	case "query":
		annotationName = "api.query"
	case "header":
		annotationName = "api.header"
	case "path":
		annotationName = "api.path"
	case "cookie":
		annotationName = "api.cookie"
	case "formData":
		annotationName = "api.form"
	case "body":
		annotationName = "api.body"
	case "raw_body":
		annotationName = "api.raw_body"
	default:
		return nil
	}

	annotationValue := name
	if lastDot := strings.LastIndex(name, "."); lastDot != -1 {
		annotationValue = name[lastDot+1:]
	}

	return &idl_ast.Annotation{
		Name:  annotationName,
		Value: &idl_ast.ConstantValue{Value: strconv.Quote(annotationValue)},
	}
}

func (c *Converter) getMainThriftFileName() string {
	base := filepath.Base(c.filePath)
	prefix := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(prefix, prefix+".thrift")
}

func (c *Converter) getOrCreateDefs(filename string) *idl_ast.Definitions {
	if _, ok := c.fileDefinitions[filename]; !ok {
		c.fileDefinitions[filename] = &idl_ast.Definitions{}
	}
	return c.fileDefinitions[filename]
}

// toLowerCamelCase converts a string to lowerCamelCase.
func toLowerCamelCase(s string) string {
	pascal := toPascalCase(s)
	if pascal == "" {
		return ""
	}
	runes := []rune(pascal)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func (c *Converter) getOutputDirPrefix() string {
	base := filepath.Base(c.filePath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// sanitizeName removes consecutively repeated words from a PascalCase string.
func sanitizeName(name string) string {
	re := regexp.MustCompile(`([A-Z]+[a-z0-9]*)`)
	parts := re.FindAllString(name, -1)
	if len(parts) <= 1 {
		return name
	}

	var cleanParts []string
	if len(parts) > 0 {
		cleanParts = append(cleanParts, parts[0])
	}

	for i := 1; i < len(parts); i++ {
		if parts[i] != parts[i-1] {
			cleanParts = append(cleanParts, parts[i])
		}
	}

	return strings.Join(cleanParts, "")
}
