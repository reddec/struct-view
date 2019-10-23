package structview

import "github.com/dave/jennifer/jen"

type CacheGen struct {
	TypeName    string
	KeyType     string
	KeyImport   string
	ValueType   string
	ValueImport string
}

func (cc *CacheGen) UpdaterType() string { return "Updater" + cc.TypeName }

func (cc *CacheGen) CacheType() string { return "cache" + cc.TypeName }

func (cc *CacheGen) Key() jen.Code {
	if cc.KeyImport == "" {
		return jen.Id(cc.KeyType)
	}
	if cc.KeyType[0] == '*' {
		return jen.Op("*").Qual(cc.KeyImport, cc.KeyType[1:])
	}
	return jen.Qual(cc.KeyImport, cc.KeyType)
}

func (cc *CacheGen) Value() jen.Code {
	if cc.ValueImport == "" {
		return jen.Id(cc.ValueType)
	}
	if cc.ValueType[0] == '*' {
		return jen.Op("*").Qual(cc.ValueImport, cc.ValueType[1:])
	}
	return jen.Qual(cc.ValueImport, cc.ValueType)
}

func (cc *CacheGen) Generate() jen.Code {
	out := jen.Empty().Add(cc.generateUpdater()).Line()
	out = out.Line().Add(cc.generateManager()).Line()
	out = out.Line().Add(cc.generateCacheNode()).Line()
	return out
}

func (cc *CacheGen) generateManager() jen.Code {
	code := jen.Func().Id("New" + cc.TypeName + "Func").Params(jen.Id("updateFunc").Id(cc.UpdaterType() + "Func")).Op("*").Id(cc.TypeName).BlockFunc(func(group *jen.Group) {
		group.Return().Id("New" + cc.TypeName).Call(jen.Id("updateFunc"))
	}).Line()

	code = code.Line().Func().Id("New" + cc.TypeName).Params(jen.Id("updater").Id(cc.UpdaterType())).Op("*").Id(cc.TypeName).BlockFunc(func(group *jen.Group) {
		group.ReturnFunc(func(ret *jen.Group) {
			ret.Op("&").Id(cc.TypeName).Values(
				jen.Id("cache").Op(":").Make(jen.Map(cc.Key()).Add(jen.Op("*").Id(cc.CacheType()))),
				jen.Id("updater").Op(":").Id("updater"),
			)
		})
	}).Line()

	code = code.Line().Type().Id(cc.TypeName).StructFunc(func(group *jen.Group) {
		group.Id("lock").Qual("sync", "RWMutex")
		group.Id("cache").Map(cc.Key()).Add(jen.Op("*").Id(cc.CacheType()))
		group.Id("updater").Id(cc.UpdaterType())
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Find").Params(jen.Id("key").Add(cc.Key())).Op("*").Id(cc.CacheType()).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("RLock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("RUnlock").Call()
		group.Return().Id("mgr").Dot("cache").Index(jen.Id("key"))
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("FindOrCreate").Params(jen.Id("key").Add(cc.Key())).Op("*").Id(cc.CacheType()).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("RLock").Call()
		group.Id("entry").Op(":=").Id("mgr").Dot("cache").Index(jen.Id("key"))
		group.If().Id("entry").Op("!=").Nil().BlockFunc(func(ok *jen.Group) {
			ok.Id("mgr").Dot("lock").Dot("RUnlock").Call()
			ok.Return().Id("entry")
		})
		group.Id("mgr").Dot("lock").Dot("RUnlock").Call()
		group.Id("mgr").Dot("lock").Dot("Lock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("Unlock").Call()
		group.Id("entry").Op("=").Id("mgr").Dot("cache").Index(jen.Id("key"))
		group.If().Id("entry").Op("!=").Nil().BlockFunc(func(ok *jen.Group) {
			ok.Return().Id("entry")
		})
		group.Id("entry").Op("=").Op("&").Id(cc.CacheType()).ValuesFunc(func(values *jen.Group) {
			values.Id("key").Op(":").Id("key")
			values.Id("updater").Op(":").Id("mgr").Dot("updater")
		})
		group.Id("mgr").Dot("cache").Index(jen.Id("key")).Op("=").Id("entry")
		group.Return().Id("entry")
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Get").Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("key").Add(cc.Key()),
	).Params(cc.Value(), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Return().Id("mgr").Dot("FindOrCreate").Call(jen.Id("key")).Dot("Ensure").Call(jen.Id("ctx"))
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Set").Params(
		jen.Id("key").Add(cc.Key()),
		jen.Id("value").Add(cc.Value()),
	).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("FindOrCreate").Call(jen.Id("key")).Dot("Set").Call(jen.Id("value"))
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Purge").Params(jen.Id("key").Add(cc.Key())).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("Lock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("Unlock").Call()
		group.Delete(jen.Id("mgr").Dot("cache"), jen.Id("key"))
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("PurgeAll").Params().BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("Lock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("Unlock").Call()
		group.Id("mgr").Dot("cache").Op("=").Make(jen.Map(cc.Key()).Op("*").Id(cc.CacheType()))
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Snapshot").Params().Map(cc.Key()).Add(cc.Value()).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("RLock").Call()
		group.Defer().Id("mgr").Dot("lock").Dot("RUnlock").Call()
		group.Id("snapshot").Op(":=").Make(jen.Map(cc.Key()).Add(cc.Value()), jen.Len(jen.Id("mgr").Dot("cache")))
		group.For().List(jen.Id("key"), jen.Id("cache")).Op(":=").Range().Id("mgr").Dot("cache").BlockFunc(func(loop *jen.Group) {
			loop.If().Op("!").Id("cache").Dot("valid").BlockFunc(func(fail *jen.Group) {
				fail.Continue()
			})
			loop.Id("snapshot").Index(jen.Id("key")).Op("=").Id("cache").Dot("data")
		})
		group.Return().Id("snapshot")
	}).Line()

	return code
}

func (cc *CacheGen) generateCacheNode() jen.Code {
	code := jen.Type().Id(cc.CacheType()).StructFunc(func(group *jen.Group) {
		group.Id("valid").Bool()
		group.Id("lock").Qual("sync", "Mutex")
		group.Id("data").Add(cc.Value())
		group.Id("key").Add(cc.Key())
		group.Id("updater").Add(jen.Id(cc.UpdaterType()))
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Valid").Params().Bool().BlockFunc(func(group *jen.Group) {
		group.Return().Id("cache").Dot("valid")
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Invalidate").Params().BlockFunc(func(group *jen.Group) {
		group.Id("cache").Dot("valid").Op("=").False()
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Key").Params().Add(cc.Key()).BlockFunc(func(group *jen.Group) {
		group.Return().Id("cache").Dot("key")
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Get").Params().Add(cc.Value()).BlockFunc(func(group *jen.Group) {
		group.Return().Id("cache").Dot("data")
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Ensure").Params(jen.Id("ctx").Qual("context", "Context")).Params(cc.Value(), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Err().Op(":=").Id("cache").Dot("Update").Call(jen.Id("ctx"), jen.False())
		group.Return(jen.Id("cache").Dot("data"), jen.Err())
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Set").Params(jen.Id("value").Add(cc.Value())).BlockFunc(func(group *jen.Group) {
		group.Id("cache").Dot("lock").Dot("Lock").Call()
		group.Defer().Id("cache").Dot("lock").Dot("Unlock").Call()
		group.Id("cache").Dot("data").Op("=").Id("value")
		group.Id("cache").Dot("valid").Op("=").True()
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Update").Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("force").Bool()).Error().BlockFunc(func(group *jen.Group) {
		group.Id("cache").Dot("lock").Dot("Lock").Call()
		group.Defer().Id("cache").Dot("lock").Dot("Unlock").Call()
		group.If(jen.Id("cache").Dot("valid").Op("&&").Op("!").Id("force")).BlockFunc(func(ok *jen.Group) {
			ok.Return().Nil()
		})
		group.Id("temp").Op(",").Err().Op(":=").Id("cache").Dot("updater").Dot("Update").Call(
			jen.Id("ctx"), jen.Id("cache").Dot("key"))
		group.If(jen.Err().Op("!=").Nil()).BlockFunc(func(fail *jen.Group) {
			fail.Return().Err()
		})
		group.Id("cache").Dot("data").Op("=").Id("temp")
		group.Id("cache").Dot("valid").Op("=").True()
		group.Return().Nil()
	}).Line()

	return code
}

func (cc *CacheGen) generateUpdater() jen.Code {
	def := jen.Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("key").Add(cc.Key())).Params(cc.Value(), jen.Error())
	code := jen.Type().Id(cc.UpdaterType()).InterfaceFunc(func(group *jen.Group) {
		group.Id("Update").Add(def)
	}).Line()
	code = code.Line().Type().Id(cc.UpdaterType() + "Func").Func().Add(def).Line()
	code = code.Line().Func().Params(jen.Id("fn").Id(cc.UpdaterType() + "Func")).Id("Update").Add(def).BlockFunc(func(group *jen.Group) {
		group.Return().Id("fn").Call(jen.Id("ctx"), jen.Id("key"))
	})
	return code
}
