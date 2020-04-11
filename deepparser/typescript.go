package deepparser

import (
	"go/ast"
	"strings"
)

type TypeScript struct {
	Typer
}

func (tsg *TypeScript) mapBase(typeName string) string {
	if strings.HasPrefix(typeName, "int") ||
		strings.HasPrefix(typeName, "float") ||
		strings.HasPrefix(typeName, "uint") ||
		typeName == "byte" {
		return "number"
	}
	if typeName == "string" {
		return "string"
	}
	if typeName == "bool" {
		return "boolean"
	}
	return typeName
}

func (tsg *TypeScript) MapType(t ast.Expr) string {
	if v, ok := t.(*ast.Ident); ok {
		return tsg.mapBase(v.Name)
	}
	if acc, ok := t.(*ast.SelectorExpr); ok {
		return acc.Sel.Name
	}
	if ptr, ok := t.(*ast.StarExpr); ok {
		return tsg.MapType(ptr.X)
	}

	if arr, ok := t.(*ast.ArrayType); ok {
		return "Array<" + tsg.MapType(arr.Elt) + ">"
	}

	return "any"
}

func (tsg *TypeScript) MapField(st *StField) string {
	tp := tsg.MapType(st.AST.Type)
	if st.Omitempty {
		return tp + " | null"
	}
	return tp
}
