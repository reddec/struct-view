package deepparser

import (
	"bytes"
	"github.com/fatih/structtag"
	"github.com/reddec/godetector"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"path/filepath"
)

type Typer struct {
	Ordered []*Definition
	Parsed  map[string]*Definition
}

func (tsg *Typer) Add(def *Definition) {
	uid := def.Import.Path + "@" + def.TypeName
	_, ok := tsg.Parsed[uid]
	if ok {
		return
	}
	if tsg.Parsed == nil {
		tsg.Parsed = make(map[string]*Definition)
	}
	tsg.Ordered = append(tsg.Ordered, def)
	def.removeJSONIgnoredFields()
	tsg.Parsed[uid] = def

	for _, f := range def.StructFields() {
		alias := detectPackageInType(f.AST.Type)
		typeName := rebuildTypeNameWithoutPackage(f.AST.Type)
		def := findDefinitionFromAst(typeName, alias, def.File, def.FileDir)

		if def != nil {
			tsg.Add(def)
		}
	}
}

func (tsg *Typer) AddFromDir(typeName string, dir string) {
	def := findDefinitionFromAst(typeName, "", nil, dir)
	if def == nil {
		return
	}
	tsg.Add(def)
}

func (tsg *Typer) AddFromFile(typeName string, filename string) {
	tsg.AddFromDir(typeName, filepath.Dir(filename))
}

func (tsg *Typer) AddFromImport(typeName string, importPath string) {
	location, err := godetector.FindPackageDefinitionDir(importPath, ".")
	if err != nil {
		return
	}
	tsg.AddFromDir(typeName, location)
}

type Definition struct {
	Import   godetector.Import
	Decl     *ast.GenDecl
	Type     *ast.TypeSpec
	TypeName string
	FS       *token.FileSet
	FileDir  string
	File     *ast.File
}

func findDefinitionFromAst(typeName, alias string, file *ast.File, fileDir string) *Definition {
	var importDef godetector.Import
	if alias != "" {
		v, err := godetector.ResolveImport(alias, file, fileDir)
		if err != nil {
			log.Println("failed resolve import for", alias, "from dir", fileDir, ":", err)
			return nil
		}
		importDef = *v
	} else {
		v, err := godetector.InspectImportByDir(fileDir)
		if err != nil {
			log.Println("failed inspect", fileDir, ":", err)
			return nil
		}
		importDef = *v
	}

	var fs token.FileSet
	importFile, err := parser.ParseDir(&fs, importDef.Location, nil, parser.AllErrors)
	if err != nil {
		log.Println("failed parse", importDef.Location, ":", err)
		return nil
	}
	for _, packageDefintion := range importFile {
		for _, packageFile := range packageDefintion.Files {
			for _, decl := range packageFile.Decls {
				if v, ok := decl.(*ast.GenDecl); ok && v.Tok == token.TYPE {
					for _, spec := range v.Specs {
						if st, ok := spec.(*ast.TypeSpec); ok && st.Name.Name == typeName {
							return &Definition{
								Import:   importDef,
								Decl:     v,
								Type:     st,
								FS:       &fs,
								TypeName: typeName,
								FileDir:  importDef.Location,
								File:     packageFile,
							}
						}
					}
				}
			}
		}
	}
	return nil
}

type StField struct {
	Name      string
	Type      string
	Tag       string
	Comment   string
	AST       *ast.Field
	Omitempty bool
}

func (def *Definition) isStruct() bool {
	_, ok := def.Type.Type.(*ast.StructType)
	return ok
}

func (def *Definition) StructFields() []*StField {
	st, ok := def.Type.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	if st.Fields == nil || len(st.Fields.List) == 0 {
		return nil
	}
	var ans []*StField
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		if !ast.IsExported(field.Names[0].Name) {
			continue
		}
		var comment string
		if field.Comment != nil {
			comment = field.Comment.Text()
		}
		f := &StField{
			Name:    field.Names[0].Name,
			Tag:     field.Names[0].Name,
			Type:    astPrint(field.Type, def.FS),
			Comment: comment,
			AST:     field,
		}
		ans = append(ans, f)
		if field.Tag == nil {
			continue
		}
		s := field.Tag.Value
		s = s[1 : len(s)-1]
		val, err := structtag.Parse(s)
		if err != nil {
			log.Println("failed parse tags:", err)
			continue
		}

		if jsTag, err := val.Get("json"); err == nil && jsTag != nil {
			if jsTag.Name != "-" {
				f.Tag = jsTag.Name
			}
			f.Omitempty = jsTag.HasOption("omitempty")
		}
	}
	return ans
}

func (def *Definition) removeJSONIgnoredFields() {
	st, ok := def.Type.Type.(*ast.StructType)
	if !ok {
		return
	}
	if st.Fields == nil || len(st.Fields.List) == 0 {
		return
	}
	var filtered []*ast.Field
	for _, field := range st.Fields.List {
		filtered = append(filtered, field)
		if field.Tag == nil {
			continue
		}
		s := field.Tag.Value
		s = s[1 : len(s)-1]
		val, err := structtag.Parse(s)
		if err != nil {
			log.Println("failed parse tags:", err)
			continue
		}
		if !ast.IsExported(field.Names[0].Name) {
			filtered = filtered[:len(filtered)-1]
			continue
		}

		if jsTag, err := val.Get("json"); err == nil && jsTag != nil {
			if jsTag.Value() == "-" {
				filtered = filtered[:len(filtered)-1]
			}
		}
	}
	st.Fields.List = filtered
}

func astPrint(t ast.Node, fs *token.FileSet) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fs, t)
	return buf.String()
}

func detectPackageInType(t ast.Expr) string {
	if acc, ok := t.(*ast.SelectorExpr); ok {
		return acc.X.(*ast.Ident).Name
	} else if ptr, ok := t.(*ast.StarExpr); ok {
		return detectPackageInType(ptr.X)
	} else if arr, ok := t.(*ast.ArrayType); ok {
		return detectPackageInType(arr.Elt)
	}
	return ""
}

func rebuildOps(t ast.Expr) string {
	if ptr, ok := t.(*ast.StarExpr); ok {
		return "*" + rebuildOps(ptr.X)
	}
	if arr, ok := t.(*ast.ArrayType); ok {
		return "[]" + rebuildOps(arr.Elt)
	}
	return ""
}

func rebuildTypeNameWithoutPackage(t ast.Expr) string {
	if v, ok := t.(*ast.Ident); ok {
		return v.Name
	}
	if ptr, ok := t.(*ast.StarExpr); ok {
		return rebuildTypeNameWithoutPackage(ptr.X)
	}
	if acc, ok := t.(*ast.SelectorExpr); ok {
		return acc.Sel.Name
	}
	if arr, ok := t.(*ast.ArrayType); ok {
		return rebuildTypeNameWithoutPackage(arr.Elt)
	}
	return ""
}
