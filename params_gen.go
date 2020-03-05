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
				varType := getVarType(file, field.Type, localPackage)
				group.Id(varName).Add(varType).Tag(map[string]string{
					"json": jsonName,
					"yaml": yamlName,
					"form": formName,
					"xml":  xmlName,
				})
				invokeVars = append(invokeVars, jen.Id(vName).Dot(varName))
			}
		}
	}).Line()

	appType := getVarType(file, fd.Recv.List[0].Type, localPackage)

	var retTypes []jen.Code
	if fd.Type.Results != nil {
		for _, ret := range fd.Type.Results.List {
			if len(ret.Names) > 0 {
				for _, name := range ret.Names {
					retTypes = append(retTypes, jen.Id(name.Name).Add(getVarType(file, ret.Type, localPackage)))
				}
			} else {
				retTypes = append(retTypes, getVarType(file, ret.Type, localPackage))
			}
		}
	}

	code.Line().Func().Params(jen.Id(vName).Op("*").Id(tName)).Id("Invoke").Params(jen.Id("app").Add(appType)).Params(retTypes...).BlockFunc(func(group *jen.Group) {
		if len(retTypes) > 0 {
			group.Return().Id("app").Dot(fd.Name.Name).Call(invokeVars...)
		} else {
			group.Id("app").Dot(fd.Name.Name).Call(invokeVars...)
		}
	})

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

func getVarType(file *ast.File, expr ast.Expr, localPackage string) *jen.Statement {
	if v, ok := expr.(*ast.Ident); ok {
		if isBuiltin(v.Name) {
			return jen.Id(v.Name)
		}
		return jen.Qual(localPackage, v.Name)
	}
	if ref, ok := expr.(*ast.StarExpr); ok {
		return jen.Op("*").Add(getVarType(file, ref.X, localPackage))
	}
	if arr, ok := expr.(*ast.ArrayType); ok {
		return jen.Index().Add(getVarType(file, arr.Elt, localPackage))
	}
	if mp, ok := expr.(*ast.MapType); ok {
		return jen.Map(getVarType(file, mp.Key, localPackage)).Add(getVarType(file, mp.Value, localPackage))
	}
	if selector, ok := expr.(*ast.SelectorExpr); ok {
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
