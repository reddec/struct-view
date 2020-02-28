package structview

import "github.com/dave/jennifer/jen"

type TimedCache struct {
	TypeName    string
	ValueType   string
	ValueImport string
	Array       bool
	PrivateInit bool
	WithEvents  bool
}

func (cc *TimedCache) UpdaterType() string { return "Updater" + cc.TypeName }

func (cc *TimedCache) EventType() string { return "Event" + cc.TypeName + "Func" }

func (cc *TimedCache) EventKindType() string { return "Event" + cc.TypeName }

func (cc *TimedCache) EventTypeName(event string) string { return "Event" + cc.TypeName + event }

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
	prefix := "New"
	if cc.PrivateInit {
		prefix = "new"
	}
	code := jen.Func().Id(prefix+cc.TypeName+"Func").Params(jen.Id("keepAlive").Qual("time", "Duration"), jen.Id("updateFunc").Id(cc.UpdaterType()+"Func")).Op("*").Id(cc.TypeName).BlockFunc(func(group *jen.Group) {
		group.Return().Id(prefix+cc.TypeName).Call(jen.Id("keepAlive"), jen.Id("updateFunc"))
	}).Line()

	code = code.Line().Func().Id(prefix+cc.TypeName).Params(jen.Id("keepAlive").Qual("time", "Duration"), jen.Id("updater").Id(cc.UpdaterType())).Op("*").Id(cc.TypeName).BlockFunc(func(group *jen.Group) {
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
		if cc.WithEvents {
			group.Id("listeners").Index().Id(cc.EventType())
		}
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Data").Params().Add(cc.Value()).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("RLock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("RUnlock").Call()
		group.Return().Id("mgr").Dot("data")
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Get").Params(jen.Id("ctx").Qual("context", "Context")).Params(cc.Value(), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Id("now").Op(":=").Qual("time", "Now").Call()
		group.Id("mgr").Dot("lock").Dot("Lock").Call()
		group.Id("delta").Op(":=").Id("now").Dot("Sub").Call(jen.Id("mgr").Dot("lastUpdate"))
		group.If().Id("delta").Op("<=").Id("mgr").Dot("keepAlive").BlockFunc(func(ok *jen.Group) {
			ok.Id("mgr").Dot("lock").Dot("Unlock").Call()
			ok.Return(jen.Id("mgr").Dot("data"), jen.Nil())
		})
		group.List(jen.Id("data"), jen.Err()).Op(":=").Id("mgr").Dot("updater").Dot("Update").Call(jen.Id("ctx"))
		group.If().Err().Op("==").Nil().BlockFunc(func(ok *jen.Group) {
			ok.Id("mgr").Dot("data").Op("=").Id("data")
			ok.Id("mgr").Dot("lastUpdate").Op("=").Id("now")
			ok.Id("mgr").Dot("lock").Dot("Unlock").Call()
			if cc.WithEvents {
				group.Id("mgr").Dot("notify").Call(jen.Id("data"), jen.Id(cc.EventTypeName("Update")))
			}
		}).Else().BlockFunc(func(fail *jen.Group) {
			fail.Id("mgr").Dot("lock").Dot("Unlock").Call()
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
		group.Id("mgr").Dot("lastUpdate").Op("=").Qual("time", "Now").Call()
		group.Id("mgr").Dot("data").Op("=").Id("data")
		group.Id("mgr").Dot("lock").Dot("Unlock").Call()
		if cc.WithEvents {
			group.Id("mgr").Dot("notify").Call(jen.Id("data"), jen.Id(cc.EventTypeName("Update")))
		}
	}).Line()

	if cc.WithEvents {
		code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Subscribe").Params(
			jen.Id("handlerFunc").Id(cc.EventType()),
		).BlockFunc(func(group *jen.Group) {
			group.Id("mgr").Dot("listeners").Op("=").Append(jen.Id("mgr").Dot("listeners"), jen.Id("handlerFunc"))
		}).Line()
		code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("notify").Params(
			jen.Id("value").Add(cc.Value()),
			jen.Id("event").Id(cc.EventKindType()),
		).BlockFunc(func(group *jen.Group) {
			group.Id("n").Op(":=").Len(jen.Id("mgr").Dot("listeners"))
			group.For(jen.Id("i").Op(":=").Lit(0), jen.Id("i").Op("<").Id("n"), jen.Id("i").Op("++")).BlockFunc(func(iter *jen.Group) {
				iter.Id("mgr").Dot("listeners").Index(jen.Id("i")).Call(
					jen.Id("value"),
					jen.Id("event"),
				)
			})
		})
	}
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
	if cc.WithEvents {
		code = code.Line().Type().Id(cc.EventType()).Func().Params(jen.Id("value").Add(cc.Value()), jen.Id("kind").Id(cc.EventKindType()))
		code = code.Line().Type().Id(cc.EventKindType()).Int()
		code = code.Line().Const().DefsFunc(func(group *jen.Group) {
			group.Id(cc.EventTypeName("Update")).Id(cc.EventKindType()).Op("=").Lit(0b01).Comment("0b01")
			group.Id(cc.EventTypeName("DeleteAll")).Id(cc.EventKindType()).Op("=").Lit(0b110).Comment("0b110")
		})
	}
	return code
}
