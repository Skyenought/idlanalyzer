# IDL AST JSON 结构文档

本文档详细描述了一种通用的 JSON 结构，用于表示从 IDL（接口描述语言，如 Thrift）源文件中解析出的抽象语法树（AST）。该结构旨在提供丰富、精确且易于机器处理的代码元数据。

## 顶层结构: `IDLSchema`

`IDLSchema` 是整个 JSON 文档的根对象，代表了一个被完整解析的 IDL 项目。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `schemaVersion` | `string` | **必需**。此 JSON 结构的语义版本号，例如 `"1.0"`。 |
| `idlType` | `string` | **必需**。原始 IDL 的类型，例如 `"thrift"` 或 `"protobuf"`。 |
| `files` | `[File]` | **必需**。一个数组，包含了项目中所有被解析的 IDL 文件的信息。 |

## `File` 对象

代表一个独立的 IDL 源文件及其完整内容。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `path` | `string` | **必需**。该文件相对于解析根目录的相对路径。 |
| `location` | `Location` | *可选*。描述整个文件在源文本中的范围。 |
| `imports` | `[Import]` | *可选*。一个数组，包含了该文件中所有的 `include` 或 `import` 语句。 |
| `syntax` | `string` | *可选*。IDL 的语法版本，例如 `"proto3"`。 |
| `definitions` | `Definitions` | **必需**。一个容器，包含了该文件中定义的所有核心元素。 |
| `namespaces` | `[Namespace]` | **必需**。一个数组，包含了该文件中所有的命名空间声明。 |
| `options` | `[Annotation]` | *可选*。用于文件级别的注解或选项。 |

## `Definitions` 对象

一个容器，用于按类型组织一个文件中的所有定义。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `services` | `[Service]` | *可选*。文件中定义的所有服务。 |
| `messages` | `[Message]` | *可选*。文件中定义的所有消息体（struct, union, exception）。 |
| `enums` | `[Enum]` | *可选*。文件中定义的所有枚举。 |
| `constants` | `[Constant]` | *可选*。文件中定义的所有常量。 |
| `typedefs` | `[Typedef]` | *可选*。文件中定义的所有类型别名。 |

---

## 核心定义类型

### `Service` 对象

代表一个 RPC 服务接口。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。服务定义之前的前导注释。 |
| `location` | `Location` | *可选*。服务定义在源文件中的精确范围。 |
| `content` | `string` | *可选*。服务定义的原始代码文本。 |
| `name` | `string` | **必需**。服务的名称。 |
| `fullyQualifiedName` | `string` | *可选*。服务的完全限定名称，格式为 `path/to/file.thrift#ServiceName`。 |
| `functions` | `[Function]` | **必需**。服务中定义的所有 RPC 方法。 |
| `extends` | `string` | *可选*。如果该服务继承了另一个服务，这里是父服务的名称。 |
| `annotations` | `[Annotation]` | *可选*。应用于服务的注解列表。 |

### `Message` 对象

代表一个结构化的数据类型，可以是 `struct`, `union`, 或 `exception`。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。消息体定义之前的前导注释。 |
| `location` | `Location` | *可选*。消息体定义在源文件中的精确范围。 |
| `content` | `string` | *可选*。消息体定义的原始代码文本。 |
| `name` | `string` | **必需**。消息体的名称。 |
| `fullyQualifiedName` | `string` | *可选*。消息体的完全限定名称，格式为 `path/to/file.thrift#MessageName`。 |
| `type` | `string` | **必需**。消息体的具体类型，值为 `"struct"`, `"union"`, 或 `"exception"`。 |
| `fields` | `[Field]` | **必需**。消息体中包含的字段列表。 |
| `annotations` | `[Annotation]` | *可选*。应用于消息体的注解列表。 |

### `Enum` 对象

代表一个枚举类型。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。枚举定义之前的前导注释。 |
| `location` | `Location` | *可选*。枚举定义在源文件中的精确范围。 |
| `content` | `string` | *可选*。枚举定义的原始代码文本。 |
| `name` | `string` | **必需**。枚举的名称。 |
| `fullyQualifiedName` | `string` | *可选*。枚举的完全限定名称，格式为 `path/to/file.thrift#EnumName`。 |
| `values` | `[EnumValue]` | **必需**。枚举中包含的所有成员。 |
| `annotations` | `[Annotation]` | *可选*。应用于枚举的注解列表。 |

### `Constant` 对象

代表一个具名常量。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。常量定义之前的前导注释。 |
| `location` | `Location` | *可选*。整个常量定义语句在源文件中的精确范围。 |
| `content` | `string` | *可选*。常量定义的原始代码文本。 |
| `name` | `string` | **必需**。常量的名称。 |
| `fullyQualifiedName` | `string` | *可选*。常量的完全限定名称，格式为 `path/to/file.thrift#ConstantName`。 |
| `type` | `Type` | **必需**。常量的数据类型。 |
| `value` | `string` | **必需**。常量值的字符串表示形式。**注意**：更复杂的常量结构请参考 `ConstantValue`。 |
| `annotations` | `[Annotation]` | *可选*。应用于常量的注解列表。 |

### `Typedef` 对象

代表一个类型别名。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。类型别名定义之前的前导注释。 |
| `location` | `Location` | *可选*。整个类型别名定义语句在源文件中的精确范围。 |
| `content` | `string` | *可选*。类型别名定义的原始代码文本。 |
| `alias` | `string` | **必需**。新定义的类型名称。 |
| `type` | `Type` | **必需**。原始的、被起别名的类型。 |
| `annotations` | `[Annotation]` | *可选*。应用于类型别名的注解列表。 |

---

## 核心构建块

### `Function` 对象

定义了服务中的一个 RPC 方法。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。函数定义之前的前导注释。 |
| `location` | `Location` | *可选*。函数定义在源文件中的精确范围。 |
| `signature` | `string` | *可选*。函数的完整签名文本。 |
| `name` | `string` | **必需**。函数的名称。 |
| `fullyQualifiedName` | `string` | *可选*。函数的完全限定名称，格式为 `path/to/file.thrift#ServiceName.FunctionName`。 |
| `returnType` | `Type` | **必需**。函数的返回类型。 |
| `parameters` | `[Field]` | **必需**。函数的参数列表。 |
| `throws` | `[Field]` | *可选*。函数可能抛出的异常列表。 |
| `annotations` | `[Annotation]` | *可选*。应用于函数的注解列表。 |

### `Field` 对象

定义了消息体中的一个字段或函数的参数。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。字段定义之前的前导注释。 |
| `location` | `Location` | *可选*。整个字段定义行在源文件中的精确范围。 |
| `id` | `integer` | **必需**。字段的唯一数字标识符。 |
| `name` | `string` | **必需**。字段的名称。 |
| `type` | `Type` | **必需**。字段的数据类型。 |
| `required` | `string` | **必需**。字段的限定符，如 `"required"`, `"optional"`。 |
| `defaultValue` | `ConstantValue` | *可选*。字段的默认值。 |
| `annotations` | `[Annotation]` | *可选*。应用于字段的注解列表。 |

### `EnumValue` 对象

定义了枚举中的一个具体成员。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。枚举成员之前的前导注释。 |
| `location` | `Location` | *可选*。枚举成员在源文件中的精确范围。 |
| `name` | `string` | **必需**。枚举成员的名称。 |
| `value` | `integer` | **必需**。枚举成员对应的整数值。 |
| `annotations` | `[Annotation]` | *可选*。应用于枚举成员的注解列表。 |

### `Type` 对象

一个可递归的结构，用于表示任何数据类型。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `location` | `Location` | *可选*。该类型引用在源文件中的精确位置。 |
| `name` | `string` | **必需**。类型的名称，如 `"string"`, `"map"`, `"UserProfile"`。 |
| `isPrimitive` | `boolean` | **必需**。如果为 `true`，表示该类型是 IDL 内置的基本类型。 |
| `fullyQualifiedName` | `string` | *可选*。非基本类型的完全限定名，格式为 `path/to/file.thrift#TypeName`。 |
| `keyType` | `Type` | *可选*。当 `name` 为 `map` 时，表示其键的类型。 |
| `valueType` | `Type` | *可选*。当 `name` 为 `map`, `list`, 或 `set` 时，表示其值的类型。 |

---

## 常量与注解

### `ConstantValue` 对象

一个可以表示任何常量值的递归结构。它可以是简单字面量、标识符、列表或 Map/Message 字面量。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `value` | `any` | **必需**。该字段的实际类型决定了它的种类：<br>- `string`: 字符串字面量 (`"hello"`) 或标识符 (`Status.OK`)<br>- `integer`: 整数 (`123`)<br>- `number`: 浮点数 (`3.14`)<br>- `boolean`: 布尔值 (`true`)<br>- `[ConstantValue]`: 列表/数组 (`[1, 2, 3]`)<br>- `[ConstantMapEntry]`: Map 或 Message (`{ "key": "value" }`) |

### `ConstantMapEntry` 对象

表示 Map 或 Message 字面量中的一个键值对。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `key` | `ConstantValue` | **必需**。字段名或 Map 的键。 |
| `value` | `ConstantValue` | **必需**。字段值或 Map 的值。 |

### `Annotation` 对象

代表一个元数据注解或选项，例如 Thrift 的 `(api.get="/hello")`。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `name` | `string` | **必需**。注解的名称，例如 `"api.get"`。 |
| `value` | `ConstantValue` | *可选*。注解的值。 |

---

## 原子结构

### `Import` 对象

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。导入语句之前的前导注释。 |
| `location` | `Location` | *可选*。整个导入语句在源文件中的精确范围。 |
| `value` | `string` | **必需**。导入路径的原始字符串，包含引号。 |
| `path` | `string` | **必需**。解析和规范化后的、相对于项目根目录的相对路径。 |

### `Namespace` 对象

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `comments` | `[Comment]` | *可选*。命名空间声明之前的前导注释。 |
| `location` | `Location` | *可选*。命名空间声明在源文件中的精确范围。 |
| `scope` | `string` | **必需**。作用的语言或范围，如 `"go"` 或 `"java"`。 |
| `name` | `string` | **必需**。命名空间的名称。 |

### `Comment` 对象

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `text` | `string` | **必需**。注释的完整文本内容。 |
| `location` | `Location` | *可选*。注释在源文件中的精确范围。 |

### `Location` 对象

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `start` | `Position` | **必需**。范围的起始位置。 |
| `end` | `Position` | **必需**。范围的结束位置。 |

### `Position` 对象

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `line` | `integer` | **必需**。1-based 的行号。 |
| `column` | `integer` | **必需**。1-based 的列号。 |
| `offset` | `integer` | **必需**。0-based 的字节偏移量。 |

---

## 程序化访问与搜索

除了作为静态的数据结构，`IDLSchema` 还提供了内置的搜索功能，允许通过编程方式高效地查询 AST。

### 核心查询函数: `FindByFQN`

-   **`FindByFQN(fqn string) []any`**:
    这是最核心的查询方法。它首先尝试**精确匹配**完全限定名称（FQN），如果找不到，则会进行**后缀匹配**。例如，使用 `"MyStruct"` 作为 `fqn` 可以找到 `path/to/file.thrift#MyStruct`。

### 便捷查询函数

为了方便使用，提供了一系列类型安全的包装函数：

-   `FindServicesByFQN(fqn string) []*Service`
-   `FindMessagesByFQN(fqn string) []*Message`
-   `FindEnumsByFQN(fqn string) []*Enum`
-   `FindConstantsByFQN(fqn string) []*Constant`
-   `FindFunctionsByFQN(fqn string) []*Function`

### 细粒度查询

还支持更具体的查询，例如只查找 `struct` 类型的 `Message`：

-   `FindStructsByFQN(fqn string) []*Message`
-   `FindUnionsByFQN(fqn string) []*Message`
-   `FindExceptionsByFQN(fqn string) []*Message`

### 辅助函数

-   **`SplitFQN(fqn string) (filePath, definitionName string, ok bool)`**:
    一个实用的辅助函数，用于从一个完全限定名称中解析出文件路径和定义名称。
    -   **示例**: `"path/to/file.thrift#MyStruct"` -> `"path/to/file.thrift"`, `"MyStruct"`, `true`