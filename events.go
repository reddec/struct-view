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
	WithBus        bool
	WithMirror     bool
	WithSink       bool
	FromMirror     bool
	FromIgnoreCase bool
	BusName        string
	MirrorType     string
	Private        bool
	Hints          map[string]string // Event->Struct Name
}

func (eg EventGenerator) Generate(directories ...string) (jen.Code, error) {
	var (
		events   []string
		types    []string
		payloads []*Struct
	)
	code := jen.Empty()
	for _, directory := range directories {
		fs := token.NewFileSet()
		p, err := parser.ParseDir(fs, directory, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		var (
			name        string
			comment     string
			prevComment string
		)
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
					var eventsToGenerate []string
					for eventName, structName := range eg.Hints {
						if name == structName {
							eventsToGenerate = append(eventsToGenerate, eventName)
						}
					}
					for _, line := range strings.Split(comment, "\n") {
						line = strings.TrimSpace(line)
						val, err := structtag.Parse(line)
						if err != nil {
							continue
						}
						if event, err := val.Get("event"); err == nil && event != nil {
							eventsToGenerate = append(eventsToGenerate, event.Name)
						}

					}
					for _, eventName := range eventsToGenerate {
						typeName := eventName
						if eg.Private {
							typeName = "event" + eventName
						}
						code.Add(eg.generateForType(info, typeName, false))
						code.Add(jen.Line())
						events = append(events, eventName)
						types = append(types, typeName)
						payloads = append(payloads, info)
					}

					comment = ""
				}
				return true
			})
		}
	}

	if eg.WithBus {
		code.Add(eg.generateBus(eg.BusName, events, types))
		code.Add(jen.Line())
	}
	if eg.WithMirror && eg.WithBus {
		code.Add(eg.generateMirrorConstructorForBus(eg.MirrorType, eg.BusName, events))
		code.Add(jen.Line())
	}
	if eg.WithSink && eg.WithBus {
		code.Add(eg.generateSinkForBus(eg.BusName, events, payloads))
		code.Add(jen.Line())
	}
	if eg.FromMirror && eg.WithBus {
		code.Add(eg.generateBusSource(eg.BusName, events, payloads))
		code.Add(jen.Line())
	}

	return code, nil
}

func (eg EventGenerator) generateForType(info *Struct, eventName string, flat bool) jen.Code {
	handlerType := jen.Func().Params(info.Qual())
	impl := eventName
	mirrorFunc := jen.Func().Params(jen.Id("eventName").String(), jen.Id("payload").Interface())
	code := jen.Type().Id(impl).StructFunc(func(group *jen.Group) {
		group.Id("lock").Qual("sync", "RWMutex")
		group.Id("handlers").Index().Add(handlerType)
		if eg.WithMirror {
			group.Id("mirror").Add(mirrorFunc)
		}
	}).Line()
	code = code.Func().Params(jen.Id("ev").Op("*").Id(impl)).Id("Subscribe").Params(jen.Id("handler").Add(handlerType)).BlockFunc(func(group *jen.Group) {
		group.Id("ev").Dot("lock").Dot("Lock").Call()
		group.Id("ev").Dot("handlers").Op("=").Append(jen.Id("ev").Dot("handlers"), jen.Id("handler"))
		group.Id("ev").Dot("lock").Dot("Unlock").Call()
	}).Line()

	code = code.Func().Params(jen.Id("ev").Op("*").Id(impl)).Id("Emit").ParamsFunc(func(params *jen.Group) {
		if len(info.Definition.Fields.List) != 0 {
			params.Id("payload").Add(info.Qual())
		}
	}).BlockFunc(func(group *jen.Group) {
		if len(info.Definition.Fields.List) == 0 {
			group.Id("payload").Op(":=").Add(info.Qual()).Values()
		}
		group.Id("ev").Dot("lock").Dot("RLock").Call()
		group.For(jen.List(jen.Id("_"), jen.Id("handler")).Op(":=").Range().Id("ev").Dot("handlers")).BlockFunc(func(iter *jen.Group) {
			iter.Id("handler").Call(jen.Id("payload"))
		})
		group.Id("ev").Dot("lock").Dot("RUnlock").Call()
		if eg.WithMirror {
			group.If(jen.Id("mirror").Op(":=").Id("ev").Dot("mirror"), jen.Id("mirror").Op("!=").Nil()).BlockFunc(func(mirror *jen.Group) {
				mirror.Id("mirror").Call(jen.Lit(eventName), jen.Id("payload"))
			})
		}
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

func (eg EventGenerator) generateBusSource(eventBus string, events []string, types []*Struct) jen.Code {
	code := jen.Func().Params(jen.Id("ev").Op("*").Id(eventBus)).Id("Emit").Params(jen.Id("eventName").String(), jen.Id("payload").Interface()).BlockFunc(func(group *jen.Group) {
		if eg.FromIgnoreCase {
			group.Switch(jen.Qual("strings", "ToUpper").Call(jen.Id("eventName"))).BlockFunc(func(sw *jen.Group) {
				for i, eventName := range events {
					eventType := types[i]
					sw.Case(jen.Lit(strings.ToUpper(eventName))).BlockFunc(func(evGroup *jen.Group) {
						evGroup.If(jen.List(jen.Id("obj"), jen.Id("ok")).Op(":=").Id("payload").Op(".").Parens(eventType.Qual()), jen.Id("ok")).BlockFunc(func(casted *jen.Group) {
							casted.Id("ev").Dot(eventName).Dot("Emit").Call(jen.Id("obj"))
						}).Else().If(jen.List(jen.Id("obj"), jen.Id("ok")).Op(":=").Id("payload").Op(".").Parens(jen.Op("*").Add(eventType.Qual())), jen.Id("ok")).BlockFunc(func(casted *jen.Group) {
							casted.Id("ev").Dot(eventName).Dot("Emit").Call(jen.Op("*").Id("obj"))
						})
					})
				}
			})
		} else {
			group.Switch(jen.Id("eventName")).BlockFunc(func(sw *jen.Group) {
				for i, eventName := range events {
					eventType := types[i]
					sw.Case(jen.Lit(eventName)).BlockFunc(func(evGroup *jen.Group) {
						evGroup.If(jen.List(jen.Id("obj"), jen.Id("ok")).Op(":=").Id("payload").Op(".").Parens(eventType.Qual()), jen.Id("ok")).BlockFunc(func(casted *jen.Group) {
							casted.Id("ev").Dot(eventName).Dot("Emit").Call(jen.Id("obj"))
						}).Else().If(jen.List(jen.Id("obj"), jen.Id("ok")).Op(":=").Id("payload").Op(".").Parens(jen.Op("*").Add(eventType.Qual())), jen.Id("ok")).BlockFunc(func(casted *jen.Group) {
							casted.Id("ev").Dot(eventName).Dot("Emit").Call(jen.Op("*").Id("obj"))
						})
					})
				}
			})
		}
	}).Line()

	code.Func().Params(jen.Id("ev").Op("*").Id(eventBus)).Id("Payload").Params(jen.Id("eventName").String()).Interface().BlockFunc(func(group *jen.Group) {
		if eg.FromIgnoreCase {
			group.Switch(jen.Qual("strings", "ToUpper").Call(jen.Id("eventName"))).BlockFunc(func(sw *jen.Group) {
				for i, eventName := range events {
					eventType := types[i]
					sw.Case(jen.Lit(strings.ToUpper(eventName))).BlockFunc(func(evGroup *jen.Group) {
						evGroup.Return().Op("&").Add(eventType.Qual()).Values()
					})
				}
			})

		} else {
			group.Switch(jen.Id("eventName")).BlockFunc(func(sw *jen.Group) {
				for i, eventName := range events {
					eventType := types[i]
					sw.Case(jen.Lit(eventName)).BlockFunc(func(evGroup *jen.Group) {
						evGroup.Return().Op("&").Add(eventType.Qual()).Values()
					})
				}
			})
		}
		group.Return().Nil()
	})
	return code
}

func (eg EventGenerator) generateMirrorConstructorForBus(emitterType, eventBus string, events []string) jen.Code {
	mirrorFunc := jen.Func().Params(jen.Id("eventName").String(), jen.Id("payload").Interface())
	return jen.Func().Id(eventBus + "WithMirror").Params(jen.Id("mirror").Add(mirrorFunc)).Op("*").Id(eventBus).BlockFunc(func(group *jen.Group) {
		group.Var().Id("bus").Id(eventBus)
		for _, eventName := range events {
			group.Id("bus").Dot(eventName).Dot("mirror").Op("=").Id("mirror")
		}
		group.Return().Op("&").Id("bus")
	})
}

func (eg EventGenerator) generateSinkForBus(eventBus string, events []string, types []*Struct) jen.Code {
	mirrorFunc := jen.Func().Params(jen.Id("eventName").String(), jen.Id("payload").Interface())
	return jen.Func().Params(jen.Id("bus").Op("*").Id(eventBus)).Id("Sink").Params(jen.Id("sink").Add(mirrorFunc)).Op("*").Id(eventBus).BlockFunc(func(group *jen.Group) {
		for i, eventName := range events {
			inType := types[i]
			group.Id("bus").Dot(eventName).Dot("Subscribe").Call(jen.Func().Params(jen.Id("payload").Add(inType.Qual())).BlockFunc(func(closure *jen.Group) {
				closure.Id("sink").Call(jen.Lit(eventName), jen.Id("payload"))
			}))
		}
		group.Return().Id("bus")
	})
}
