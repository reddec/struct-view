package structview

import (
	"github.com/dave/jennifer/jen"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
)

type BinaryGenerator struct {
	TypeName string
}

func (bg BinaryGenerator) Generate(directory string) (jen.Code, string, error) {
	fs := token.NewFileSet()
	p, err := parser.ParseDir(fs, directory, nil, parser.ParseComments)
	if err != nil {
		return nil, "", err
	}
	var (
		name string
		pack string
	)
	out := jen.Empty()
	for _, def := range p {
		ast.Inspect(def, func(node ast.Node) bool {
			switch v := node.(type) {
			case *ast.Package:
				pack = v.Name
			case *ast.TypeSpec:
				name = v.Name.Name
			case *ast.StructType:
				info, err := WrapStruct(directory, name, v)
				if err != nil {
					log.Println(err)
					return true
				}
				if info.Struct == bg.TypeName {
					header, _, fields := bg.generateConstants(info)
					out.Add(header).Line()
					out.Line().Add(bg.generateMarshaller(info, fields)).Line()
					out.Line().Add(bg.generateUnMarshaller(info, fields)).Line()
					return false
				}
			}
			return true
		})
	}
	return out, pack, nil
}

func (bg BinaryGenerator) generateConstants(structType *Struct) (code jen.Code, bufferSize int, fields []*ast.Field) {
	for _, field := range structType.Definition.Fields.List {
		if !ast.IsExported(field.Names[0].Name) {
			continue
		}
		if _, isRef := field.Type.(*ast.StarExpr); isRef {
			continue
		}
		ident, isSimple := field.Type.(*ast.Ident)
		if !isSimple {
			continue
		}
		typeName := ident.Name

		switch typeName {
		case "uint8":
			bufferSize += 1
		case "uint16":
			bufferSize += 2
		case "uint32":
			bufferSize += 4
		case "uint64":
			bufferSize += 8
		default:
			continue
		}
		fields = append(fields, field)
	}
	code = jen.Const().Id(structType.Struct + "BinarySize").Op("=").Lit(bufferSize)
	return
}

func (bg BinaryGenerator) generateMarshaller(structType *Struct, fields []*ast.Field) jen.Code {

	return jen.Func().Params(jen.Id("data").Op("*").Id(structType.Struct)).Id("Marshal").Params(jen.Id("stream").Qual("io", "Writer")).Error().BlockFunc(func(group *jen.Group) {
		group.List(jen.Id("buffer"), jen.Err()).Op(":=").Id("data").Dot("MarshalBinary").Call()
		group.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
			failed.Return().Err()
		})
		group.List(jen.Id("n"), jen.Err()).Op(":=").Id("stream").Dot("Write").Call(jen.Id("buffer").Index(jen.Empty(), jen.Empty()))
		group.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
			failed.Return().Err()
		})
		group.If().Id("n").Op("!=").Id(structType.Struct + "BinarySize").BlockFunc(func(failed *jen.Group) {
			failed.Return().Qual("io", "ErrShortWrite")
		})
		group.Return().Nil()
	}).Line().Line().Func().Params(jen.Id("data").Op("*").Id(structType.Struct)).Id("MarshalBinary").Params().Params(jen.Index().Byte(), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Var().Id("buffer").Index(jen.Id(structType.Struct + "BinarySize")).Byte()
		var offset int
		for _, field := range fields {
			typeName := field.Type.(*ast.Ident).Name

			switch typeName {
			case "uint8":
				group.Id("buffer").Index(jen.Lit(offset)).Op("=").Id("data").Dot(field.Names[0].Name)
				offset += 1
			case "uint16":
				group.Qual("binary", "BigEndian").Dot("PutUint16").Call(jen.Id("buffer").Index(jen.Lit(offset), jen.Lit(offset+2)), jen.Id("data").Dot(field.Names[0].Name))
				offset += 2
			case "uint32":
				group.Qual("binary", "BigEndian").Dot("PutUint32").Call(jen.Id("buffer").Index(jen.Lit(offset), jen.Lit(offset+4)), jen.Id("data").Dot(field.Names[0].Name))
				offset += 4
			case "uint64":
				group.Qual("binary", "BigEndian").Dot("PutUint64").Call(jen.Id("buffer").Index(jen.Lit(offset), jen.Lit(offset+8)), jen.Id("data").Dot(field.Names[0].Name))
				offset += 8
			}
		}
		group.Return(jen.Id("buffer").Index(jen.Empty(), jen.Empty()), jen.Nil())
	})
}

func (bg BinaryGenerator) generateUnMarshaller(structType *Struct, fields []*ast.Field) jen.Code {

	return jen.Func().Params(jen.Id("data").Op("*").Id(structType.Struct)).Id("Unmarshal").Params(jen.Id("stream").Qual("io", "Reader")).Error().BlockFunc(func(group *jen.Group) {
		group.Var().Id("buffer").Index(jen.Id(structType.Struct + "BinarySize")).Byte()
		group.List(jen.Id("n"), jen.Err()).Op(":=").Id("stream").Dot("Reader").Call(jen.Id("buffer").Index(jen.Empty(), jen.Empty()))
		group.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
			failed.Return().Err()
		})
		group.If().Id("n").Op("!=").Id(structType.Struct + "BinarySize").BlockFunc(func(failed *jen.Group) {
			failed.Return().Qual("io", "ErrShortRead")
		})
		group.Return().Id("data").Dot("UnmarshalBinary").Call(jen.Id("buffer").Index(jen.Empty(), jen.Empty()))
	}).Line().Line().Func().Params(jen.Id("data").Op("*").Id(structType.Struct)).Id("UnmarshalBinary").Params(jen.Id("buffer").Index().Byte()).Error().BlockFunc(func(group *jen.Group) {
		group.If().Len(jen.Id("buffer")).Op("<").Id(structType.Struct + "BinarySize").BlockFunc(func(failed *jen.Group) {
			failed.Return().Qual("fmt", "Errorf").Call(jen.Lit("too small buffer to decode "+structType.Struct+": required at least %v, but got %v"), jen.Id(structType.Struct+"BinarySize"), jen.Len(jen.Id("buffer")))
		})
		var offset int
		for _, field := range fields {
			typeName := field.Type.(*ast.Ident).Name
			switch typeName {
			case "uint8":
				group.Id("data").Dot(field.Names[0].Name).Op("=").Id("buffer").Index(jen.Lit(offset))
				offset += 1
			case "uint16":
				group.Id("data").Dot(field.Names[0].Name).Op("=").Qual("binary", "BigEndian").Dot("Uint16").Call(jen.Id("buffer").Index(jen.Lit(offset), jen.Lit(offset+2)))
				offset += 2
			case "uint32":
				group.Id("data").Dot(field.Names[0].Name).Op("=").Qual("binary", "BigEndian").Dot("Uint32").Call(jen.Id("buffer").Index(jen.Lit(offset), jen.Lit(offset+4)))
				offset += 4
			case "uint64":
				group.Id("data").Dot(field.Names[0].Name).Op("=").Qual("binary", "BigEndian").Dot("Uint64").Call(jen.Id("buffer").Index(jen.Lit(offset), jen.Lit(offset+8)))
				offset += 8
			}
		}
		group.Return().Nil()
	})
}
