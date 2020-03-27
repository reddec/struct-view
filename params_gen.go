package structview

import (
	"fmt"
	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	"github.com/reddec/godetector"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

type ParamsGen struct {
	Dir        string
	StructName string
	Gin        bool
}

func (pg *ParamsGen) Generate() (jen.Code, error) {
	var fs token.FileSet
	parsed, err := parser.ParseDir(&fs, pg.Dir, nil, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("failed parse %s: %w", pg.Dir, err)
	}

	localPackage, err := godetector.FindImportPath(pg.Dir)
	if err != nil {
		return nil, fmt.Errorf("detect package for %s: %w", pg.Dir, err)
	}

	code := jen.Empty()
	for _, pkg := range parsed {
		for _, file := range pkg.Files {
			for _, dec := range file.Decls {
				code.Add(pg.checkFuncDecl(file, dec, localPackage)).Line()
			}
		}

	}
	return code, nil
}

func (pg *ParamsGen) checkFuncDecl(file *ast.File, decl ast.Decl, localPackage string) jen.Code {
	code := jen.Empty()
	fd, ok := decl.(*ast.FuncDecl)
	if !ok || fd.Recv == nil {
		return code
	}
	if !ast.IsExported(fd.Name.Name) {
		return code
	}

	for _, f := range fd.Recv.List {
		if pg.validReceiver(f) {
			code.Line().Add(pg.handleFunction(file, fd, localPackage))
			break
		}
	}
	return code
}

func (pg *ParamsGen) handleFunction(file *ast.File, fd *ast.FuncDecl, localPackage string) jen.Code {
	tName := fd.Name.Name + "Params"

	var syms []string
	for _, w := range strings.Split(strcase.ToSnake(tName), "_") {
		syms = append(syms, w[:1])
	}
	vName := strings.Join(syms, "")
	var invokeVars []jen.Code
	code := jen.Type().Id(tName).StructFunc(func(group *jen.Group) {
		if fd.Type.Params == nil {
			return
		}
		for _, field := range fd.Type.Params.List {
			for _, name := range field.Names {
				varName := strcase.ToCamel(name.Name)
				jsonName := strcase.ToSnake(name.Name)
				xmlName := jsonName
				yamlName := jsonName
				formName := jsonName
				pathName := jsonName
				varType := TypeDefinition(file, field.Type, localPackage)
				group.Id(varName).Add(varType).Tag(map[string]string{
					"json": jsonName,
					"yaml": yamlName,
					"form": formName,
					"xml":  xmlName,
					"path": pathName,
				})
				invokeVars = append(invokeVars, jen.Id(vName).Dot(varName))
			}
		}
	}).Line()

	appType := TypeDefinition(file, fd.Recv.List[0].Type, localPackage)

	var retTypes []jen.Code
	if fd.Type.Results != nil {
		for _, ret := range fd.Type.Results.List {
			if len(ret.Names) > 0 {
				for _, name := range ret.Names {
					retTypes = append(retTypes, jen.Id(name.Name).Add(TypeDefinition(file, ret.Type, localPackage)))
				}
			} else {
				retTypes = append(retTypes, TypeDefinition(file, ret.Type, localPackage))
			}
		}
	}

	code.Line().Func().Params(jen.Id(vName).Op("*").Id(tName)).Id("Invoke").Params(jen.Id("app").Add(appType)).Params(retTypes...).BlockFunc(func(group *jen.Group) {
		if len(retTypes) > 0 {
			group.Return().Id("app").Dot(fd.Name.Name).Call(invokeVars...)
		} else {
			group.Id("app").Dot(fd.Name.Name).Call(invokeVars...)
		}
	}).Line()

	if pg.Gin {
		code.Line().Func().Params(jen.Id(vName).Op("*").Id(tName)).Id("Bind").Params(jen.Id("gctx").Op("*").Qual("github.com/gin-gonic/gin", "Context")).Error().BlockFunc(func(group *jen.Group) {
			group.If(jen.Err().Op(":=").Id("gctx").Dot("Bind").Call(jen.Id(vName)), jen.Err().Op("!=").Nil()).BlockFunc(func(fail *jen.Group) {
				fail.Return().Err()
			})
			group.If(jen.Err().Op(":=").Id("gctx").Dot("BindUri").Call(jen.Id(vName)), jen.Err().Op("!=").Nil()).BlockFunc(func(fail *jen.Group) {
				fail.Return().Err()
			})
			group.Return().Nil()
		}).Line()
	}

	return code
}

func (pg *ParamsGen) validReceiver(f *ast.Field) bool {
	return pg.isStructType(f.Type)
}

func (pg *ParamsGen) isStructType(expr ast.Expr) bool {
	if st, ok := expr.(*ast.StarExpr); ok {
		return pg.isStructType(st.X)
	}
	if id, ok := expr.(*ast.Ident); ok {
		return pg.StructName == id.Name
	}

	return false
}

func TypeDefinition(file *ast.File, typeDef ast.Expr, localPackage string) *jen.Statement {
	if v, ok := typeDef.(*ast.Ident); ok {
		if isBuiltin(v.Name) {
			return jen.Id(v.Name)
		}
		return jen.Qual(localPackage, v.Name)
	}
	if ref, ok := typeDef.(*ast.StarExpr); ok {
		return jen.Op("*").Add(TypeDefinition(file, ref.X, localPackage))
	}
	if arr, ok := typeDef.(*ast.ArrayType); ok {
		return jen.Index().Add(TypeDefinition(file, arr.Elt, localPackage))
	}
	if mp, ok := typeDef.(*ast.MapType); ok {
		return jen.Map(TypeDefinition(file, mp.Key, localPackage)).Add(TypeDefinition(file, mp.Value, localPackage))
	}
	if selector, ok := typeDef.(*ast.SelectorExpr); ok {
		return jen.Qual(findImportPath(file, selector.X.(*ast.Ident).Name), selector.Sel.Name)
	}
	return jen.Empty()
}

func findImportPath(file *ast.File, pkg string) string {
	for _, imp := range file.Imports {
		if imp.Name != nil {
			if imp.Name.Name == pkg {
				path, _ := strconv.Unquote(imp.Path.Value)
				return path
			}
		}

	}
	for _, imp := range file.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if godetector.FindPackageNameByDir(path) == pkg {
			return path
		}
	}
	return ""
}

func isBuiltin(name string) bool {
	for _, k := range types.Typ {
		if k.Name() == name {
			return true
		}
	}
	if name == "error" {
		return true
	}
	return false
}
