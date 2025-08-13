package idl_ast

import "fmt"

// -----------------------------------------------------------------------------
// 新增的、用于表示复杂常量值的核心结构
// -----------------------------------------------------------------------------

// ConstantMapEntry 表示 Map 或 Message 字面量中的一个键值对。
// 例如：`title: "example"`
type ConstantMapEntry struct {
	// Key 是字段名或 Map 的键。在 Protobuf 中它是一个标识符，在 Thrift 中它可以是任何常量值。
	// 为了通用，我们这里用 ConstantValue。
	Key   *ConstantValue `json:"key"`
	Value *ConstantValue `json:"value"`
}

// ConstantValue 是一个可以表示任何常量值的递归结构。
// 它可以是简单字面量、标识符、列表，或 Map/Message 字面量。
type ConstantValue struct {
	// Value 字段的实际类型决定了它的种类：
	// - string: 字符串字面量 (例如 "hello") 或 标识符 (例如 Status.OK)
	// - int64: 整数
	// - float64: 浮点数
	// - bool:布尔值
	// - []*ConstantValue: 列表/数组 (例如 [1, 2, 3])
	// - []*ConstantMapEntry: Map 或 Message (例如 { "key": "value", info: {...} })
	Value any `json:"value"`
}

func (cv *ConstantValue) StringValue() (string, error) {
	if cv == nil || cv.Value == nil {
		return "", fmt.Errorf("constant value is nil")
	}

	// 例如 "hello" 会被存储为 `"hello"`。
	str, ok := cv.Value.(string)
	if !ok {
		return "", fmt.Errorf("constant value is not a string, but %T", cv.Value)
	}

	// 移除字符串两端的引号
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		return str[1 : len(str)-1], nil
	}

	// 也处理单引号的情况
	if len(str) >= 2 && str[0] == '\'' && str[len(str)-1] == '\'' {
		return str[1 : len(str)-1], nil
	}

	// 如果值是字符串但没有引号（不符合 transformer 的预期，但做个兜底），
	// 就按原样返回，并认为这是一个非法的字符串字面量。
	return str, fmt.Errorf("value is a string but not properly quoted: %s", str)
}

func (cv *ConstantValue) MustStringValueWithQuote() string {
	ret, err := cv.StringValue()
	if err != nil {
		panic(err)
	}
	return ret
}

// Annotation 代表一个元数据注解或选项。
// 例如：Thrift 的 `(api.get="/hello")` 或 Protobuf 的 `option (api.get) = "/hello";`
type Annotation struct {
	Name  string         `json:"name"` // 例如 "api.get" 或 "(google.api.http).get"
	Value *ConstantValue `json:"value,omitempty"`
}

// -----------------------------------------------------------------------------
// 核心 AST 结构定义 (已更新)
// -----------------------------------------------------------------------------

// IDLSchema 是整个 JSON 结构的根对象，代表了一个被完整解析的 IDL 项目。
type IDLSchema struct {
	SchemaVersion string `json:"schemaVersion"`
	IDLType       string `json:"idlType"` // 例如 "thrift" 或 "protobuf"
	Files         []File `json:"files"`
}

// File 代表一个独立的 IDL 文件及其完整内容。
type File struct {
	Path        string       `json:"path"`
	Location    *Location    `json:"location,omitempty"`
	Imports     []Import     `json:"imports,omitempty"`
	Syntax      string       `json:"syntax,omitempty"`
	Definitions Definitions  `json:"definitions"`
	Namespaces  []Namespace  `json:"namespaces"`
	Options     []Annotation `json:"options,omitempty"` // 用于文件级选项
}

// Definitions 是一个容器，用于组织一个文件中定义的所有不同类型的元素。
type Definitions struct {
	Services  []Service  `json:"services,omitempty"`
	Messages  []Message  `json:"messages,omitempty"`
	Enums     []Enum     `json:"enums,omitempty"`
	Constants []Constant `json:"constants,omitempty"`
	Typedefs  []Typedef  `json:"typedefs,omitempty"`
}

// Import 代表一条导入语句，如 'include "shared.thrift"'。
type Import struct {
	Comments []Comment `json:"comments,omitempty"`
	Location *Location `json:"location,omitempty"`
	Value    string    `json:"value"` // 包含引号的原始路径
	Path     string    `json:"path"`  // 解析和规范化后的路径
}

// Namespace 定义了特定语言的代码生成命名空间或包。
type Namespace struct {
	Comments []Comment `json:"comments,omitempty"`
	Location *Location `json:"location,omitempty"`
	Scope    string    `json:"scope"`
	Name     string    `json:"name"`
}

// -----------------------------------------------------------------------------
// 顶层定义 (已更新)
// -----------------------------------------------------------------------------

// Service 定义了一个 RPC 服务接口。
type Service struct {
	Comments           []Comment    `json:"comments,omitempty"`
	Location           *Location    `json:"location,omitempty"`
	Content            string       `json:"content,omitempty"`
	Name               string       `json:"name"`
	FullyQualifiedName string       `json:"fullyQualifiedName,omitempty"`
	Functions          []Function   `json:"functions"`
	Extends            string       `json:"extends,omitempty"`
	Annotations        []Annotation `json:"annotations,omitempty"`
}

// Message 代表一个结构化的数据类型，可以是 struct, union, 或 exception。
type Message struct {
	Comments           []Comment    `json:"comments,omitempty"`
	Location           *Location    `json:"location,omitempty"`
	Content            string       `json:"content,omitempty"`
	Name               string       `json:"name"`
	FullyQualifiedName string       `json:"fullyQualifiedName,omitempty"`
	Type               string       `json:"type"` // "struct", "union", "exception"
	Fields             []Field      `json:"fields"`
	Annotations        []Annotation `json:"annotations,omitempty"`
}

// Enum 定义了一个枚举类型。
type Enum struct {
	Comments           []Comment    `json:"comments,omitempty"`
	Location           *Location    `json:"location,omitempty"`
	Content            string       `json:"content,omitempty"`
	Name               string       `json:"name"`
	FullyQualifiedName string       `json:"fullyQualifiedName,omitempty"`
	Values             []EnumValue  `json:"values"`
	Annotations        []Annotation `json:"annotations,omitempty"`
}

// Constant 定义了一个具名常量。
type Constant struct {
	Comments           []Comment    `json:"comments,omitempty"`
	Location           *Location    `json:"location,omitempty"`
	Content            string       `json:"content,omitempty"`
	Name               string       `json:"name"`
	FullyQualifiedName string       `json:"fullyQualifiedName,omitempty"`
	Type               Type         `json:"type"`
	Value              string       `json:"value"` // 值的字符串表示 (为了简单兼容旧逻辑，未来可替换为 ConstantValue)
	Annotations        []Annotation `json:"annotations,omitempty"`
}

// Typedef 定义了一个类型别名。
type Typedef struct {
	Comments    []Comment    `json:"comments,omitempty"`
	Location    *Location    `json:"location,omitempty"`
	Content     string       `json:"content,omitempty"`
	Alias       string       `json:"alias"`
	Type        Type         `json:"type"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

// -----------------------------------------------------------------------------
// 子级定义和核心构建块 (已更新)
// -----------------------------------------------------------------------------

// Function 定义了服务中的一个 RPC 方法。
type Function struct {
	Comments           []Comment    `json:"comments,omitempty"`
	Location           *Location    `json:"location,omitempty"`
	Signature          string       `json:"signature,omitempty"`
	Name               string       `json:"name"`
	FullyQualifiedName string       `json:"fullyQualifiedName,omitempty"`
	ReturnType         Type         `json:"returnType"`
	Parameters         []Field      `json:"parameters"`
	Throws             []Field      `json:"throws,omitempty"`
	Annotations        []Annotation `json:"annotations,omitempty"`
}

// Field 定义了消息体中的一个字段或函数的参数。
type Field struct {
	Comments     []Comment      `json:"comments,omitempty"`
	Location     *Location      `json:"location,omitempty"`
	ID           int            `json:"id"`
	Name         string         `json:"name"`
	Type         Type           `json:"type"`
	Required     string         `json:"required"`
	DefaultValue *ConstantValue `json:"defaultValue,omitempty"`
	Annotations  []Annotation   `json:"annotations,omitempty"`
}

// EnumValue 定义了枚举中的一个具体成员。
type EnumValue struct {
	Comments    []Comment    `json:"comments,omitempty"`
	Location    *Location    `json:"location,omitempty"`
	Name        string       `json:"name"`
	Value       int          `json:"value"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

// Type 是一个可递归的结构，用于表示任何数据类型。
type Type struct {
	Location           *Location `json:"location,omitempty"`
	Name               string    `json:"name"`
	IsPrimitive        bool      `json:"isPrimitive"`
	FullyQualifiedName string    `json:"fullyQualifiedName,omitempty"`
	KeyType            *Type     `json:"keyType,omitempty"`
	ValueType          *Type     `json:"valueType,omitempty"`
}

// -----------------------------------------------------------------------------
// 原子结构
// -----------------------------------------------------------------------------

// Position 定义了源文件中的一个精确点（行、列、偏移量）。
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Offset int `json:"offset"`
}

// Location 定义了源文件中的一个文本范围，由起始和结束位置构成。
type Location struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Comment 代表一条源代码注释及其位置。
type Comment struct {
	Text     string    `json:"text"`
	Location *Location `json:"location,omitempty"`
}
