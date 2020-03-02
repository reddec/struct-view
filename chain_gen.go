package structview

import "github.com/dave/jennifer/jen"

type ChainGen struct {
	ContextType string
	Import      string
	TypeName    string
}

func (cg *ChainGen) Value() jen.Code {
	if cg.Import == "" {
		return jen.Id(cg.ContextType)
	}
	if cg.ContextType[0] == '*' {
		return jen.Op("*").Qual(cg.Import, cg.ContextType[1:])
	}
	return jen.Qual(cg.Import, cg.ContextType)
}

func (cg *ChainGen) Builder() string     { return cg.TypeName }
func (cg *ChainGen) Chain() string       { return cg.TypeName + "Context" }
func (cg *ChainGen) Handler() string     { return cg.TypeName + "HandlerFunc" }
func (cg *ChainGen) HandlerType() string { return cg.TypeName + "Handler" }
func (cg *ChainGen) BuilderFunc() *jen.Statement {
	return jen.Line().Func().Params(jen.Id("pipe").Op("*").Id(cg.Builder()))
}

func (cg *ChainGen) cFunc(root *jen.Statement) *jen.Statement {
	return root.Line().Func().Params(jen.Id("chain").Op("*").Id(cg.Chain()))
}

func (cg *ChainGen) Generate() jen.Code {
	code := jen.Empty()
	code.Add(cg.generateBuilder()).Line()
	code.Add(cg.generateChain()).Line()
	code.Add(cg.generateHandler()).Line()
	return code
}

func (cg *ChainGen) generateBuilder() jen.Code {
	code := jen.Type().Id(cg.Builder()).StructFunc(func(group *jen.Group) {
		group.Id("mutex").Qual("sync", "RWMutex")
		group.Id("handlers").Index().Id(cg.HandlerType())
	}).Line()
	code = code.Add(cg.BuilderFunc()).Id("To").Params(jen.Id("handlers").Op("...").Id(cg.HandlerType())).Op("*").Id(cg.Builder()).BlockFunc(func(group *jen.Group) {
		group.Id("pipe").Dot("mutex").Dot("Lock").Call()
		group.Defer().Id("pipe").Dot("mutex").Dot("Unlock").Call()
		group.Id("pipe").Dot("handlers").Op("=").Append(jen.Id("pipe").Dot("handlers"), jen.Id("handlers").Op("..."))
		group.Return().Id("pipe")
	}).Line()
	code = code.Add(cg.BuilderFunc()).Id("Use").Params(jen.Id("handlers").Op("...").Id(cg.Handler())).Op("*").Id(cg.Builder()).BlockFunc(func(group *jen.Group) {
		group.Id("pipe").Dot("mutex").Dot("Lock").Call()
		group.Defer().Id("pipe").Dot("mutex").Dot("Unlock").Call()
		group.For().List(jen.Id("_"), jen.Id("fn")).Op(":=").Range().Id("handlers").BlockFunc(func(iter *jen.Group) {
			iter.Id("pipe").Dot("handlers").Op("=").Append(jen.Id("pipe").Dot("handlers"), jen.Id(cg.HandlerType()).Call(jen.Id("fn")))
		})
		group.Return().Id("pipe")
	}).Line()
	code = code.Add(cg.BuilderFunc()).Id("Create").Params(jen.Id("data").Add(cg.Value())).Op("*").Id(cg.Chain()).BlockFunc(func(group *jen.Group) {
		group.Id("pipe").Dot("mutex").Dot("RLock").Call()
		group.Defer().Id("pipe").Dot("mutex").Dot("RUnlock").Call()
		group.Id("cp").Op(":=").Make(jen.Index().Id(cg.HandlerType()), jen.Len(jen.Id("pipe").Dot("handlers")))
		group.Copy(jen.Id("cp"), jen.Id("pipe").Dot("handlers"))
		group.Id("chain").Op(":=").Op("&").Id(cg.Chain()).ValuesFunc(func(vals *jen.Group) {
			vals.Id("handlers").Op(":").Id("cp")
			vals.Id("Data").Op(":").Id("data")
		})
		group.Return().Id("chain")
	}).Line()
	code = code.Add(cg.BuilderFunc()).Id("Run").Params(jen.Id("data").Add(cg.Value())).Error().BlockFunc(func(group *jen.Group) {
		group.Return().Id("pipe").Dot("Create").Call(jen.Id("data")).Dot("Next").Call()
	}).Line()
	code = code.Add(cg.BuilderFunc()).Id("Process").Params(jen.Id("ctx").Op("*").Id(cg.Chain())).Error().BlockFunc(func(group *jen.Group) {
		group.Err().Op(":=").Id("pipe").Dot("Run").Call(jen.Id("ctx").Dot("Data"))
		group.If().Err().Op("!=").Nil().BlockFunc(func(fail *jen.Group) {
			fail.Return().Err()
		})
		group.Return().Id("ctx").Dot("Next").Call()
	}).Line()
	code = code.Add(cg.BuilderFunc()).Id("Clone").Params().Op("*").Id(cg.Builder()).BlockFunc(func(group *jen.Group) {
		group.Id("pipe").Dot("mutex").Dot("RLock").Call()
		group.Defer().Id("pipe").Dot("mutex").Dot("RUnlock").Call()
		group.Id("cp").Op(":=").Make(jen.Index().Id(cg.HandlerType()), jen.Len(jen.Id("pipe").Dot("handlers")))
		group.Copy(jen.Id("cp"), jen.Id("pipe").Dot("handlers"))
		group.Return().Op("&").Id(cg.TypeName).ValuesFunc(func(vals *jen.Group) {
			vals.Id("handlers").Op(":").Id("cp")
		})
	}).Line()
	return code
}

func (cg *ChainGen) generateChain() jen.Code {
	code := jen.Type().Id(cg.Chain()).StructFunc(func(group *jen.Group) {
		group.Id("idx").Int()
		group.Id("handlers").Index().Id(cg.HandlerType())
		group.Id("Data").Add(cg.Value())
	}).Line()

	cg.cFunc(code).Id("Next").Params().Error().BlockFunc(func(group *jen.Group) {
		group.If().Id("chain").Dot("idx").Op(">=").Len(jen.Id("chain").Dot("handlers")).BlockFunc(func(stop *jen.Group) {
			stop.Return().Nil()
		})
		group.Id("idx").Op(":=").Id("chain").Dot("idx")
		group.Id("chain").Dot("idx").Op("++")
		group.Return().Id("chain").Dot("handlers").Index(jen.Id("idx")).Dot("Process").Call(jen.Id("chain"))
	}).Line()
	return code
}

func (cg *ChainGen) generateHandler() jen.Code {
	code := jen.Type().Id(cg.Handler()).Func().Params(jen.Id("ctx").Op("*").Id(cg.Chain())).Error().Line()
	code = code.Line().Type().Id(cg.HandlerType()).InterfaceFunc(func(group *jen.Group) {
		group.Id("Process").Params(jen.Id("ctx").Op("*").Id(cg.Chain())).Error()
	}).Line()
	code = code.Line().Func().Params(jen.Id("method").Id(cg.Handler())).Id("Process").Params(jen.Id("ctx").Op("*").Id(cg.Chain())).Error().BlockFunc(func(group *jen.Group) {
		group.Return().Id("method").Call(jen.Id("ctx"))
	}).Line()
	return code
}
