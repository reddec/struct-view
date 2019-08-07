package structview

import (
	"bufio"
	"errors"
	"github.com/dave/jennifer/jen"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func isVendorPackage(path string) (string, bool) {
	path = filepath.Join(path, "go.mod")
	if fs, err := os.Stat(path); err != nil {
		return "", false
	} else if fs.IsDir() {
		return "", false
	}
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanWords)
	if !(scanner.Scan() && scanner.Scan()) {
		return "", false
	}
	pkg := scanner.Text()
	return pkg, true
}

func isRootPackage(path string) bool {
	GOPATH := filepath.Join(os.Getenv("GOPATH"), "src")
	GOROOT := filepath.Join(os.Getenv("GOROOT"), "src")
	return isRootOf(path, GOPATH) || isRootOf(path, GOROOT)
}

func isRootOf(path, root string) bool {
	root, _ = filepath.Abs(root)
	path, _ = filepath.Abs(path)
	return root == path
}
func FindPackage(dir string) (string, error) {
	dir, _ = filepath.Abs(dir)
	return findPackage(dir)
}

func findPackage(dir string) (string, error) {
	if dir == "" {
		return "", os.ErrNotExist
	}
	if isRootPackage(dir) {
		return "", nil
	}
	pkg, ok := isVendorPackage(dir)
	if ok {
		return pkg, nil
	}
	mod := filepath.Base(dir)
	top, err := findPackage(filepath.Dir(dir))
	if err != nil {
		return "", err
	}
	if top != "" {
		return top + "/" + mod, nil
	}
	return mod, nil
}

type Struct struct {
	Struct     string
	Dir        string
	Definition *ast.StructType
	ImportPath string
}

func (s Struct) Qual() jen.Code {
	if s.ImportPath == "" {
		return jen.Id(s.Struct)
	}
	return jen.Qual(s.ImportPath, s.Struct)
}

func (s *Struct) FindClosetField(name string, containsSearch bool) *ast.Field {
	// as-is
	for _, f := range s.Definition.Fields.List {
		if name == f.Names[0].Name {
			return f
		}
	}
	// case insensitive
	for _, f := range s.Definition.Fields.List {
		if strings.EqualFold(name, f.Names[0].Name) {
			return f
		}
	}
	if containsSearch {
		// contains sensitive
		for _, f := range s.Definition.Fields.List {
			if strings.Contains(name, f.Names[0].Name) || strings.Contains(f.Names[0].Name, name) {
				return f
			}
		}
		// contains insensitive
		name = strings.ToUpper(name)
		for _, f := range s.Definition.Fields.List {
			fname := strings.ToUpper(f.Names[0].Name)
			if strings.Contains(name, fname) || strings.Contains(fname, name) {
				return f
			}
		}
	}
	return nil
}

type ToConvert struct {
	Source         Struct
	Target         Struct
	FnName         string
	SearchContains bool
	Remap          map[string]string
}

func (config ToConvert) Convert() Mapping {
	srcQual := config.Source.Qual()
	trgQual := config.Target.Qual()
	fnName := config.FnName
	var missmatch int
	code := jen.Func().Id(fnName).Params(jen.Id("src").Op("*").Add(srcQual)).Op("*").Add(trgQual).BlockFunc(func(converter *jen.Group) {
		converter.Id("dst").Op(":=").Op("&").Add(trgQual).Values()
		for _, srcField := range config.Source.Definition.Fields.List {
			var destField *ast.Field

			tName := srcField.Names[0].Name
			if newName, ok := config.Remap[tName]; ok {
				tName = newName
			}

			destField = config.Target.FindClosetField(tName, config.SearchContains)

			if destField == nil {
				log.Println("no suitable field", tName, "in", config.Target.Struct, "from", config.Source.Struct)
				missmatch++
				continue
			}

			sFieldName := srcField.Names[0].Name
			tFieldName := destField.Names[0].Name

			_, srcPtr := srcField.Type.(*ast.StarExpr)
			_, trgPtr := destField.Type.(*ast.StarExpr)

			if srcPtr && !trgPtr {
				// from pointer to non-pointer
				converter.If().Id("src").Dot(sFieldName).Op("!=").Nil().BlockFunc(func(nonNil *jen.Group) {
					nonNil.Id("dst").Dot(tFieldName).Op("=").Op("*").Id("src").Dot(sFieldName)
				})
			} else if !srcPtr && trgPtr {
				// non-pointer to pointer
				converter.Id("dst").Dot(tFieldName).Op("=").Op("&").Id("src").Dot(sFieldName)
			} else {
				converter.Id("dst").Dot(tFieldName).Op("=").Id("src").Dot(sFieldName)
			}
		}
		converter.Return().Id("dst")
	})
	return Mapping{
		Code:       code,
		NotMatched: missmatch,
	}
}

type Mapping struct {
	Code       jen.Code
	NotMatched int
}

func LoadStruct(dir, structName string) (*Struct, error) {
	fs := token.NewFileSet()
	p, err := parser.ParseDir(fs, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var name string
	var ans *Struct
	for _, def := range p {
		ast.Inspect(def, func(node ast.Node) bool {
			switch v := node.(type) {
			case *ast.TypeSpec:
				name = v.Name.Name
			case *ast.StructType:
				if name == structName {
					ans = &Struct{
						Struct:     name,
						Definition: v,
						Dir:        dir,
					}
					return false
				}
			}
			return true
		})
	}
	if ans != nil {
		pkg, err := FindPackage(dir)
		if err != nil {
			return nil, err
		}
		ans.ImportPath = pkg
		return ans, nil
	}
	return nil, errors.New("struct " + structName + " not found")
}

func WrapStruct(dir string, name string, definition *ast.StructType) (*Struct, error) {
	ans := &Struct{
		Struct:     name,
		Definition: definition,
		Dir:        dir,
	}
	pkg, err := FindPackage(dir)
	if err != nil {
		return nil, err
	}
	ans.ImportPath = pkg
	return ans, nil
}
