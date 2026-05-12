package schemair

type Registry struct {
	Namespace string
	Files     []File
}

type File struct {
	Path      string
	Package   string
	GoPackage string
	Messages  []Message
}

type Message struct {
	Name        string
	Description string
	Fields      []Field
	Enums       []Enum
}

type Field struct {
	Name        string
	Number      int
	Type        TypeRef
	Repeated    bool
	Optional    bool
	Required    bool
	Description string
}

type Enum struct {
	Name   string
	Values []EnumValue
}

type EnumValue struct {
	Name     string
	Original string
	Number   int
}

type TypeRef struct {
	Scalar  string
	Message string
	Enum    string
}
