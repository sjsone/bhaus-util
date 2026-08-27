package lsp

// BuiltInTypes is the canonical set of built-in BHaus types.
// Used by hover, completion and definition handlers to skip lookups
// for types that have no user-defined definition.
var BuiltInTypes = map[string]bool{
	// Simple types
	"String": true, "Int": true, "Integer": true, "Float": true, "Boolean": true, "Bool": true,
	"Character": true, "Char": true,
	// Prefixes
	"UnsignedInteger": true, "UInt": true, "UnsignedFloat": true, "UFloat": true,
	// Array
	"Array": true,
	// Bits<N>
	"Bits": true,
	// Special
	"Unknown": true, "True": true, "False": true,
}

// IsBuiltInType checks if a word is a built-in BHaus type.
func IsBuiltInType(word string) bool {
	return BuiltInTypes[word]
}
