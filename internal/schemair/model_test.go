package schemair

import "testing"

func TestModelContract(t *testing.T) {
	var _ string = Registry{}.Namespace
	var _ []File = Registry{}.Files
	var _ string = File{}.Path
	var _ string = File{}.Package
	var _ string = File{}.GoPackage
	var _ []Message = File{}.Messages
	var _ string = Message{}.Name
	var _ []Field = Message{}.Fields
	var _ []Enum = Message{}.Enums
	var _ int = Field{}.Number
	var _ bool = Field{}.Optional
	var _ bool = Field{}.Required
	var _ []EnumValue = Enum{}.Values
	var _ string = EnumValue{}.Original
	var _ string = TypeRef{}.Scalar
	var _ string = TypeRef{}.Message
	var _ string = TypeRef{}.Enum
}
