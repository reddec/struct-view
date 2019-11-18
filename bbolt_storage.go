package structview

import (
	"errors"
	"github.com/dave/jennifer/jen"
	"github.com/fatih/structtag"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strconv"
)

type BBoltStorage struct {
	Name       string
	TypeName   string
	Compressed bool
}

func (bbs *BBoltStorage) Render(directories ...string) (jen.Code, error) {
	code := jen.Empty()

	for _, directory := range directories {
		fs := token.NewFileSet()
		p, err := parser.ParseDir(fs, directory, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		var (
			name string
		)
		for _, def := range p {
			ast.Inspect(def, func(node ast.Node) bool {
				switch v := node.(type) {
				case *ast.TypeSpec:
					name = v.Name.Name
				case *ast.StructType:
					if name == bbs.Name {
						info, err := WrapStruct(directory, name, v)
						if err != nil {
							log.Println(err)
							return true
						}
						var primaryField *ast.Field
						var fields []field
						for _, f := range info.Definition.Fields.List {
							if f.Tag == nil || f.Tag.Value == "" {
								continue
							}
							value, err := strconv.Unquote(f.Tag.Value)
							if err != nil {
								continue
							}
							tags, err := structtag.Parse(value)
							if err != nil {
								continue
							}
							params, err := tags.Get("bbolt")
							if err != nil {
								continue
							}
							if params.HasOption("primary") {
								primaryField = f
							}
							fields = append(fields, field{
								Name:       f.Names[0].Name,
								Definition: f,
								Tag:        params,
							})
						}

						if primaryField == nil {
							log.Println("no primary field")
							return true
						}
						code.Add(bbs.renderType(info)).Line()
						code.Line().Add(bbs.renderSaver(info, primaryField, fields)).Line()
						for _, field := range fields {
							mcode, err := bbs.renderFinder(info, field, primaryField)
							if err != nil {
								log.Println(err)
								return false
							}
							code.Line().Add(mcode).Line()
						}
						code.Line().Add(bbs.renderUtils(info, primaryField))
						return false
					}
				}
				return true
			})
		}
	}

	return code, nil
}

func (bbs *BBoltStorage) renderType(dataType *Struct) jen.Code {
	return jen.Func().Id("New"+bbs.TypeName).Params(jen.Id("filename").String()).Params(jen.Op("*").Id(bbs.TypeName), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.List(jen.Id("db"), jen.Err()).Op(":=").Qual("github.com/etcd-io/bbolt", "Open").Call(jen.Id("filename"), jen.Lit(0755), jen.Nil())
		group.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
			failed.Return(jen.Nil(), jen.Err())
		})
		group.Return(jen.Op("&").Id(bbs.TypeName).ValuesFunc(func(values *jen.Group) {
			values.Id("db").Op(":").Id("db")
		}), jen.Nil())
	}).Line().Line().Type().Id(bbs.TypeName).StructFunc(func(group *jen.Group) {
		group.Id("db").Op("*").Qual("github.com/etcd-io/bbolt", "DB")
	}).Line().Line().Add(bbs.proc("Close").Params().Error().BlockFunc(func(group *jen.Group) {
		group.Return().Id("storage").Dot("db").Dot("Close").Call()
	}))
}

func (bbs *BBoltStorage) renderSaver(dataType *Struct, primaryField *ast.Field, fields []field) jen.Code {
	return bbs.proc("Save").Params(jen.Id("items").Op("...").Op("*").Add(dataType.Qual())).Error().BlockFunc(func(group *jen.Group) {
		group.Return().Id("storage").Dot("db").Dot("Update").Call(jen.Func().Params(jen.Id("tx").Op("*").Qual("github.com/etcd-io/bbolt", "Tx")).Error().BlockFunc(func(updFunc *jen.Group) {
			for _, field := range fields {
				updFunc.List(jen.Id("bucketBy"+field.Name), jen.Err()).Op(":=").Id("tx").Dot("CreateBucketIfNotExists").Call(jen.Index().Byte().Call(jen.Lit(field.Name)))
				updFunc.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
					failed.Return().Err()
				})
			}
			updFunc.For().List(jen.Id("_"), jen.Id("item")).Op(":=").Range().Id("items").BlockFunc(func(iter *jen.Group) {

				iter.List(jen.Id("data"), jen.Err()).Op(":=").Id("storage").Dot("marshal" + dataType.Struct).Call(jen.Id("item"))
				iter.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
					failed.Return().Err()
				})
				for _, field := range fields {
					if field.Definition == primaryField {
						iter.Add(field.getConverter("pk", "item."+field.Name))
						break
					}
				}

				for _, field := range fields {
					if field.isPrimary() {
						iter.Comment("save entity by primary key " + field.Name)
						iter.Err().Op("=").Id("bucketBy"+field.Name).Dot("Put").Call(jen.Id("pk"), jen.Id("data"))
						iter.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
							failed.Return().Err()
						})
					} else if field.isUnique() {
						iter.Comment("unique index by " + field.Name)
						iter.BlockFunc(func(putter *jen.Group) {
							putter.Add(field.getConverter("key", "item."+field.Name))
							putter.Err().Op("=").Id("bucketBy"+field.Name).Dot("Put").Call(jen.Id("key"), jen.Id("pk"))
							putter.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
								failed.Return().Err()
							})
						})
					} else if !field.isRange() {
						iter.Comment("non-unique index by " + field.Name)
						iter.BlockFunc(func(putter *jen.Group) {
							putter.Add(field.getConverter("key", "item."+field.Name))
							putter.List(jen.Id("list"), jen.Err()).Op(":=").Id("bucketBy" + field.Name).Dot("CreateBucketIfNotExists").Call(jen.Id("key"))
							putter.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
								failed.Return().Err()
							})
							putter.Err().Op("=").Id("list").Dot("Put").Call(jen.Id("pk"), jen.Id("pk"))
							putter.If().Err().Op("!=").Nil().BlockFunc(func(failed *jen.Group) {
								failed.Return().Err()
							})
						})
					}
				}
			})
			updFunc.Return().Nil()
		}))
	})
}

func (bbs *BBoltStorage) renderFinder(dataType *Struct, field field, primaryField *ast.Field) (jen.Code, error) {
	params := field.Tag
	var (
		unique    = params.HasOption("unique") || params.HasOption("primary")
		isPrimary = params.HasOption("primary")
		inRange   = params.HasOption("range")
	)
	if field.isRef() {
		return nil, errors.New("nullable (pointers) indexes not yet supported")
	}
	var isTime = field.isTime()
	if !isTime && field.isComplex() {
		return nil, errors.New("only simple types or time.Time as index supported")
	}

	var converter = field.getConverter("sKey", "key")

	if unique {
		return bbs.findIndexed(dataType, field.Definition, converter, isTime, isPrimary, primaryField), nil
	} else if !inRange {
		return bbs.listIndex(dataType, field.Definition, converter, isTime, isPrimary, primaryField), nil
	} else {
		return jen.Empty(), nil
	}
}

func (bbs *BBoltStorage) findIndexed(dataType *Struct, field *ast.Field, converter jen.Code, isTime, isPrimary bool, primaryField *ast.Field) jen.Code {
	var name = "GetBy" + field.Names[0].Name

	var param = jen.Id("key")
	if isTime {
		param = param.Qual("time", "Time")
	} else {
		param = param.Id(field.Type.(*ast.Ident).Name)
	}

	var result = jen.Params(jen.Id("result").Op("*").Add(dataType.Qual()), jen.Err().Error())
	return bbs.proc(name).Params(param).Add(result).BlockFunc(func(group *jen.Group) {
		group.Add(converter)
		group.Err().Op("=").Id("storage").Dot("db").Dot("View").Call(jen.Func().Params(jen.Id("tx").Op("*").Qual("github.com/etcd-io/bbolt", "Tx")).Error().BlockFunc(func(viewFunc *jen.Group) {
			if !isPrimary {
				viewFunc.List(jen.Id("result"), jen.Err()).Op("=").Id("storage").Dot("get"+dataType.Struct+"ByReference").Call(jen.Id("tx"), jen.Id("sKey"), jen.Lit(field.Names[0].Name))
			} else {
				viewFunc.List(jen.Id("result"), jen.Err()).Op("=").Id("storage").Dot("get"+dataType.Struct+"ByPrimaryKey").Call(jen.Id("tx"), jen.Id("sKey"))
			}
			viewFunc.Return().Err()
		}))
		group.Return()

	})
}

func (bbs *BBoltStorage) listIndex(dataType *Struct, field *ast.Field, converter jen.Code, isTime, isPrimary bool, primaryField *ast.Field) jen.Code {
	var name = "ListBy" + field.Names[0].Name

	var param = jen.Id("key")
	if isTime {
		param = param.Qual("time", "Time")
	} else {
		param = param.Id(field.Type.(*ast.Ident).Name)
	}

	var result = jen.Params(jen.Index().Op("*").Add(dataType.Qual()), jen.Error())
	return bbs.proc(name).Params(param).Add(result).BlockFunc(func(group *jen.Group) {
		group.Add(converter)
		group.Var().Id("result").Index().Op("*").Add(dataType.Qual())
		group.Return(
			jen.Id("result"),
			jen.Id("storage").Dot("db").Dot("View").Call(jen.Func().Params(jen.Id("tx").Qual("github.com/etcd-io/bbolt", "Tx")).Error().BlockFunc(func(viewFunc *jen.Group) {
				viewFunc.List(jen.Id("result"), jen.Err()).Op(":=").Id("storage").Dot("list"+dataType.Struct+"Index").Call(jen.Id("tx"), jen.Id("sKey"), jen.Lit(field.Names[0].Name))
				viewFunc.Return().Err()
			})),
		)

	})
}

func (bbs *BBoltStorage) renderUtils(dataType *Struct, primaryField *ast.Field) jen.Code {
	code := bbs.proc("unmarshal"+dataType.Struct).Params(jen.Id("data").Index().Byte()).Params(jen.Op("*").Add(dataType.Qual()), jen.Error()).BlockFunc(func(group *jen.Group) {
		if !bbs.Compressed {
			group.Id("reader").Op(":=").Qual("bytes", "NewReader").Call(jen.Id("data"))
		} else {
			group.List(jen.Id("reader"), jen.Err()).Op(":=").Qual("compress/gzip", "NewReader").Call(
				jen.Qual("bytes", "NewReader").Call(jen.Id("data")),
			)
			group.If().Err().Op("!=").Nil().Block(jen.Return(jen.Nil(), jen.Err()))
		}
		group.Var().Id("ans").Add(dataType.Qual())
		group.Return(jen.Op("&").Id("ans"), jen.Qual("encoding/json", "NewDecoder").Call(jen.Id("reader")).Dot("Decode").Call(jen.Op("&").Id("ans")))
	}).Line()

	code.Line().Add(bbs.proc("marshal"+dataType.Struct)).Params(jen.Id("item").Op("*").Add(dataType.Qual())).Params(jen.Index().Byte(), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Id("buf").Op(":=").Op("&").Qual("bytes", "Buffer").Values()
		if bbs.Compressed {
			group.Id("writer").Op(":=").Qual("compress/gzip", "NewWriter").Call(jen.Id("buf"))
			group.Err().Op(":=").Qual("encoding/json", "NewEncoder").Call(jen.Id("writer")).Dot("Encode").Call(jen.Id("item"))
			group.Id("writer").Dot("Close").Call()
		} else {
			group.Err().Op(":=").Qual("encoding/json", "NewEncoder").Call(jen.Id("buf")).Dot("Encode").Call(jen.Id("item"))
		}
		group.Return(jen.Id("buf").Dot("Bytes").Call(), jen.Err())
	}).Line()

	code.Line().Add(bbs.proc("get"+dataType.Struct+"ByPrimaryKey").Params(jen.Id("tx").Op("*").Qual("github.com/etcd-io/bbolt", "Tx"), jen.Id("bKey").Index().Byte())).Params(jen.Op("*").Add(dataType.Qual()), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Id("bucket").Op(":=").Id("tx").Dot("Bucket").Call(jen.Index().Byte().Call(jen.Lit(primaryField.Names[0].Name)))
		group.If().Id("bucket").Op("==").Nil().BlockFunc(func(fail *jen.Group) {
			fail.Return(jen.Nil(), jen.Qual("errors", "New").Call(jen.Lit("record not exists")))
		})

		group.Id("data").Op(":=").Id("bucket").Dot("Get").Call(jen.Id("bKey"))
		group.If().Id("data").Op("==").Nil().BlockFunc(func(fail *jen.Group) {
			fail.Return(jen.Nil(), jen.Qual("errors", "New").Call(jen.Lit("record not exists")))
		})
		group.Return().Id("storage").Dot("unmarshal" + dataType.Struct).Call(jen.Id("data"))
	}).Line()

	code.Line().Add(bbs.proc("get"+dataType.Struct+"ByReference").Params(jen.Id("tx").Op("*").Qual("github.com/etcd-io/bbolt", "Tx"), jen.Id("bKey").Index().Byte(), jen.Id("indexName").String())).Params(jen.Op("*").Add(dataType.Qual()), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Id("bucket").Op(":=").Id("tx").Dot("Bucket").Call(jen.Index().Byte().Call(jen.Id("indexName")))
		group.If().Id("bucket").Op("==").Nil().BlockFunc(func(fail *jen.Group) {
			fail.Return(jen.Nil(), jen.Qual("errors", "New").Call(jen.Lit("index not exists")))
		})
		group.Id("ref").Op(":=").Id("bucket").Dot("Get").Call(jen.Id("bKey"))
		group.If().Id("ref").Op("==").Nil().BlockFunc(func(fail *jen.Group) {
			fail.Return(jen.Nil(), jen.Qual("errors", "New").Call(jen.Lit("reference not exists")))
		})
		group.Return().Id("storage").Dot("get"+dataType.Struct+"ByPrimaryKey").Call(jen.Id("tx"), jen.Id("ref"))
	}).Line()

	code.Line().Add(bbs.proc("list"+dataType.Struct+"Index").Params(
		jen.Id("tx").Op("*").Qual("github.com/etcd-io/bbolt", "Tx"),
		jen.Id("key").Index().Byte(),
		jen.Id("indexName").String(),
	)).Params(jen.Index().Op("*").Add(dataType.Qual()), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Id("refBucket").Op(":=").Id("tx").Dot("Bucket").Call(jen.Index().Byte().Call(jen.Id("indexName")))
		group.If().Id("refBucket").Op("==").Nil().BlockFunc(func(fail *jen.Group) {
			fail.Return(jen.Nil(), jen.Nil())
		})
		group.Id("refValueBucket").Op(":=").Id("refBucket").Dot("Bucket").Call(jen.Id("key"))
		group.If().Id("refValueBucket").Op("==").Nil().BlockFunc(func(fail *jen.Group) {
			fail.Return(jen.Nil(), jen.Nil())
		})
		group.Id("bucket").Op(":=").Id("tx").Dot("Bucket").Call(jen.Index().Byte().Call(jen.Lit(primaryField.Names[0].Name)))
		group.If().Id("bucket").Op("==").Nil().BlockFunc(func(fail *jen.Group) {
			fail.Return(jen.Nil(), jen.Nil())
		})
		group.Var().Id("result").Index().Op("*").Add(dataType.Qual())
		group.Id("cursor").Op(":=").Id("refValueBucket").Dot("Cursor").Call()
		group.For(
			jen.List(jen.Id("k"), jen.Id("pk")).Op(":=").Id("cursor").Dot("First").Call(),
			jen.Id("k").Op("!=").Nil(),
			jen.List(jen.Id("k"), jen.Id("pk")).Op("=").Id("cursor").Dot("Next").Call()).BlockFunc(func(iter *jen.Group) {

			iter.Id("data").Op(":=").Id("bucket").Dot("Get").Call(jen.Id("pk"))
			iter.If().Id("data").Op("==").Nil().BlockFunc(func(fail *jen.Group) {
				fail.Return(jen.Nil(), jen.Qual("errors", "New").Call(jen.Lit("indexed record not exists")))
			})
			iter.List(jen.Id("item"), jen.Err()).Op(":=").Id("storage").Dot("unmarshal" + dataType.Struct).Call(jen.Id("data"))
			iter.If().Err().Op("!=").Nil().BlockFunc(func(fail *jen.Group) {
				fail.Return(jen.Nil(), jen.Err())
			})
			iter.Id("result").Op("=").Append(jen.Id("result"), jen.Id("item"))
		})
		group.Return(jen.Id("result"), jen.Nil())

	}).Line()

	return code
}

func (bbs *BBoltStorage) proc(name string) *jen.Statement {
	return jen.Func().Params(jen.Id("storage").Op("*").Id(bbs.TypeName)).Id(name)
}

type field struct {
	Name       string
	Definition *ast.Field
	Tag        *structtag.Tag
}

func (fd *field) isRef() bool {
	_, ok := fd.Definition.Type.(*ast.StarExpr)
	return ok
}

func (fd *field) isTime() bool {
	if selector, ok := fd.Definition.Type.(*ast.SelectorExpr); ok {
		return selector.Sel.Name == "Time" && selector.X.(*ast.Ident).Name == "time"
	}
	return false
}

func (fd *field) plainTypeName() string {
	return fd.Definition.Type.(*ast.Ident).Name
}

func (fd *field) isComplex() bool {
	if _, ok := fd.Definition.Type.(*ast.SelectorExpr); ok {
		return true
	}
	return false
}

func (fd *field) getConverter(sKey, key string) jen.Code {
	var numVar = jen.Id(key)
	// number based

	if fd.isTime() {
		numVar = jen.Id(key).Dot("UTC").Call().Dot("UnixNano").Call()
	} else if typeName := fd.plainTypeName(); typeName == "string" {
		return jen.Id(sKey).Op(":=").Index().Byte().Call(jen.Id(key))
	}
	code := jen.Empty()
	code.Id("buf").Op(":=").Op("&").Qual("bytes", "Buffer").Values().Line()
	code.Qual("binary", "Write").Call(jen.Id("buf"), jen.Qual("binary", "BigEndian"), jen.Add(numVar)).Line()
	code.Id(sKey).Op(":=").Id("buf").Dot("Bytes").Call()

	return code //jen.Index().Byte().Call(jen.Id(sKey).Op(":=").Qual("fmt", "Sprint").Call(jen.Id(key)))
}

func (fd *field) isUnique() bool {
	return fd.Tag.HasOption("unique") || fd.Tag.HasOption("primary")
}

func (fd *field) isPrimary() bool {
	return fd.Tag.HasOption("primary")
}

func (fd *field) isRange() bool {
	return fd.Tag.HasOption("range")
}
