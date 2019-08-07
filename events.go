package structview

import (
	"github.com/dave/jennifer/jen"
	"github.com/fatih/structtag"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strings"
)

type EventGenerator struct {
	WithBus bool
	BusName string
	Private bool
}

func (eg EventGenerator) Generate(directory string) (jen.Code, error) {
	fs := token.NewFileSet()
	p, err := parser.ParseDir(fs, directory, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var name string
	var comment string
	var prevComment string

	code := jen.Empty()

	var events []string
	var types []string

	for _, def := range p {
		ast.Inspect(def, func(node ast.Node) bool {
			switch v := node.(type) {
			case *ast.CommentGroup:
				prevComment = v.Text()
			case *ast.TypeSpec:
				name = v.Name.Name
				comment = strings.TrimSpace(prevComment)
				prevComment = ""
			case *ast.StructType:
				info, err := WrapStruct(directory, name, v)
				if err != nil {
					log.Println(err)
					return true
				}
				for _, line := range strings.Split(comment, "\n") {
					line = strings.TrimSpace(line)
					val, err := structtag.Parse(line)
					if err != nil {
						continue
					}

					if event, err := val.Get("event"); err == nil && event != nil {
						typeName := event.Name
						if eg.Private {
							typeName = "event" + typeName
						}
						code.Add(eg.generateForType(info, typeName))
						code.Add(jen.Line())
						events = append(events, event.Name)
						types = append(types, typeName)
					}
				}
				comment = ""
			}
			return true
		})
	}

	if eg.WithBus {
		code.Add(eg.generateBus(eg.BusName, events, types))
		code.Add(jen.Line())
	}

	return code, nil
}

func (eg EventGenerator) generateForType(info *Struct, eventName string) jen.Code {
	handlerType := jen.Func().Params(info.Qual())
	impl := eventName

	code := jen.Type().Id(impl).StructFunc(func(group *jen.Group) {
		group.Id("lock").Qual("sync", "RWMutex")
		group.Id("handlers").Index().Add(handlerType)
	}).Line()
	code = code.Func().Params(jen.Id("ev").Op("*").Id(impl)).Id("Subscribe").Params(jen.Id("handler").Add(handlerType)).BlockFunc(func(group *jen.Group) {
		group.Id("ev").Dot("lock").Dot("Lock").Call()
		group.Id("ev").Dot("handlers").Op("=").Append(jen.Id("ev").Dot("handlers"), jen.Id("handler"))
		group.Id("ev").Dot("lock").Dot("Unlock").Call()
	}).Line()
	code = code.Func().Params(jen.Id("ev").Op("*").Id(impl)).Id("Emit").Params(jen.Id("payload").Add(info.Qual())).BlockFunc(func(group *jen.Group) {
		group.Id("ev").Dot("lock").Dot("RLock").Call()
		group.For(jen.List(jen.Id("_"), jen.Id("handler")).Op(":=").Range().Id("ev").Dot("handlers")).BlockFunc(func(iter *jen.Group) {
			iter.Id("handler").Call(jen.Id("payload"))
		})
		group.Id("ev").Dot("lock").Dot("RUnlock").Call()
	}).Line()

	return code
}

func (eg EventGenerator) generateBus(typeName string, events, types []string) jen.Code {
	return jen.Type().Id(typeName).StructFunc(func(group *jen.Group) {
		for i, event := range events {
			group.Id(event).Id(types[i])
		}
	})
}
