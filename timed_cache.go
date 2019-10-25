package structview

import "github.com/dave/jennifer/jen"

type TimedCache struct {
	TypeName    string
	ValueType   string
	ValueImport string
	Array       bool
}

func (cc *TimedCache) UpdaterType() string { return "Updater" + cc.TypeName }

func (cc *TimedCache) Value() jen.Code {
	tp := jen.Empty()
	if cc.Array {
		tp = tp.Index()
	}
	if cc.ValueImport == "" {
		return tp.Id(cc.ValueType)
	}
	if cc.ValueType[0] == '*' {
		return tp.Op("*").Qual(cc.ValueImport, cc.ValueType[1:])
	}
	return tp.Qual(cc.ValueImport, cc.ValueType)
}

func (cc *TimedCache) Generate() jen.Code {
	code := jen.Empty()
	code.Add(cc.generateUpdater()).Line()
	code.Add(cc.generateManager()).Line()
	return code
}

func (cc *TimedCache) generateManager() jen.Code {
	code := jen.Func().Id("New"+cc.TypeName+"Func").Params(jen.Id("keepAlive").Qual("time", "Duration"), jen.Id("updateFunc").Id(cc.UpdaterType()+"Func")).Op("*").Id(cc.TypeName).BlockFunc(func(group *jen.Group) {
		group.Return().Id("New"+cc.TypeName).Call(jen.Id("keepAlive"), jen.Id("updateFunc"))
	}).Line()

	code = code.Line().Func().Id("New"+cc.TypeName).Params(jen.Id("keepAlive").Qual("time", "Duration"), jen.Id("updater").Id(cc.UpdaterType())).Op("*").Id(cc.TypeName).BlockFunc(func(group *jen.Group) {
		group.ReturnFunc(func(ret *jen.Group) {
			ret.Op("&").Id(cc.TypeName).Values(
				jen.Id("keepAlive").Op(":").Id("keepAlive"),
				jen.Id("updater").Op(":").Id("updater"),
			)
		})
	}).Line()

	code = code.Type().Id(cc.TypeName).StructFunc(func(group *jen.Group) {
		group.Id("lock").Qual("sync", "RWMutex")
		group.Id("lastUpdate").Qual("time", "Time")
		group.Id("keepAlive").Qual("time", "Duration")
		group.Id("data").Add(cc.Value())
		group.Id("updater").Add(jen.Id(cc.UpdaterType()))
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Get").Params().Add(cc.Value()).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("RLock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("RUnlock").Call()
		group.Return().Id("mgr").Dot("data")
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Ensure").Params(jen.Id("ctx").Qual("context", "Context")).Params(cc.Value(), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Id("now").Op(":=").Qual("time", "Now").Call()
		group.Id("mgr").Dot("lock").Dot("Lock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("Unlock").Call()
		group.Id("delta").Op(":=").Id("mgr").Dot("lastUpdate").Dot("Sub").Call(jen.Id("now"))
		group.If().Id("delta").Op("<=").Id("mgr").Dot("keepAlive").BlockFunc(func(ok *jen.Group) {
			ok.Return(jen.Id("mgr").Dot("data"), jen.Nil())
		})
		group.List(jen.Id("data"), jen.Err()).Op(":=").Id("mgr").Dot("updater").Dot("Update").Call(jen.Id("ctx"))
		group.If().Err().Op("==").Nil().BlockFunc(func(ok *jen.Group) {
			ok.Id("mgr").Dot("data").Op("=").Id("data")
			ok.Id("mgr").Dot("lastUpdate").Op("=").Id("now")
		})
		group.Return(jen.Id("mgr").Dot("data"), jen.Err())
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Invalidate").Params().BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("Lock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("Unlock").Call()
		group.Id("mgr").Dot("lastUpdate").Op("=").Qual("time", "Time").Values()
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Set").Params(jen.Id("data").Add(cc.Value())).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("Lock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("Unlock").Call()
		group.Id("mgr").Dot("lastUpdate").Op("=").Qual("time", "Now").Call()
		group.Id("mgr").Dot("data").Op("=").Id("data")
	}).Line()
	return code
}

func (cc *TimedCache) generateUpdater() jen.Code {
	def := jen.Params(jen.Id("ctx").Qual("context", "Context")).Params(cc.Value(), jen.Error())
	code := jen.Type().Id(cc.UpdaterType()).InterfaceFunc(func(group *jen.Group) {
		group.Id("Update").Add(def)
	}).Line()
	code = code.Line().Type().Id(cc.UpdaterType() + "Func").Func().Add(def).Line()
	code = code.Line().Func().Params(jen.Id("fn").Id(cc.UpdaterType() + "Func")).Id("Update").Add(def).BlockFunc(func(group *jen.Group) {
		group.Return().Id("fn").Call(jen.Id("ctx"))
	})
	return code
}
