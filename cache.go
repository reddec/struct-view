package structview

import "github.com/dave/jennifer/jen"

type CacheGen struct {
	TypeName       string
	KeyType        string
	KeyImport      string
	ValueType      string
	ValueImport    string
	PrivateInit    bool
	WithEvents     bool
	WithExpiration bool
}

func (cc *CacheGen) UpdaterType() string { return "Updater" + cc.TypeName }

func (cc *CacheGen) EventType() string { return "Event" + cc.TypeName + "Func" }

func (cc *CacheGen) EventKindType() string { return "Event" + cc.TypeName }

func (cc *CacheGen) EventTypeName(event string) string { return "Event" + cc.TypeName + event }

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
	prefix := "New"
	if cc.PrivateInit {
		prefix = "new"
	}
	code := jen.Func().Id(prefix + cc.TypeName + "Func").ParamsFunc(func(params *jen.Group) {
		if cc.WithExpiration {
			params.Id("keepAlive").Qual("time", "Duration")
		}
		params.Id("updateFunc").Id(cc.UpdaterType() + "Func")
	}).Op("*").Id(cc.TypeName).BlockFunc(func(group *jen.Group) {
		group.Return().Id(prefix + cc.TypeName).CallFunc(func(call *jen.Group) {
			if cc.WithExpiration {
				call.Id("keepAlive")
			}
			call.Id("updateFunc")
		})
	}).Line()

	code = code.Line().Func().Id(prefix + cc.TypeName).ParamsFunc(func(params *jen.Group) {
		if cc.WithExpiration {
			params.Id("keepAlive").Qual("time", "Duration")
		}
		params.Id("updater").Id(cc.UpdaterType())
	}).Op("*").Id(cc.TypeName).BlockFunc(func(group *jen.Group) {
		group.ReturnFunc(func(ret *jen.Group) {
			ret.Op("&").Id(cc.TypeName).ValuesFunc(func(values *jen.Group) {
				values.Id("cache").Op(":").Make(jen.Map(cc.Key()).Add(jen.Op("*").Id(cc.CacheType())))
				values.Id("updater").Op(":").Id("updater")
				if cc.WithExpiration {
					values.Id("keepAlive").Op(":").Id("keepAlive")
				}
			})
		})
	}).Line()

	code = code.Line().Type().Id(cc.TypeName).StructFunc(func(group *jen.Group) {
		group.Id("lock").Qual("sync", "RWMutex")
		group.Id("cache").Map(cc.Key()).Add(jen.Op("*").Id(cc.CacheType()))
		group.Id("updater").Id(cc.UpdaterType())
		if cc.WithEvents {
			group.Id("listeners").Index().Id(cc.EventType())
		}
		if cc.WithExpiration {
			group.Id("keepAlive").Qual("time", "Duration")
		}
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
			if cc.WithEvents {
				values.Id("store").Op(":").Id("mgr")
			}
			if cc.WithExpiration {
				values.Id("keepAlive").Op(":").Id("mgr").Dot("keepAlive")
			}
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

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Update").Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("key").Add(cc.Key()),
	).Params(cc.Value(), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Return().Id("mgr").Dot("FindOrCreate").Call(jen.Id("key")).Dot("Force").Call(jen.Id("ctx"))
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("UpdateAll").Params(
		jen.Id("ctx").Qual("context", "Context"),
	).Error().BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("RLock").Call()
		group.Id("ids").Op(":=").Make(jen.Index().Add(cc.Key()), jen.Lit(0), jen.Len(jen.Id("mgr").Dot("cache")))
		group.For().Id("key").Op(":=").Range().Id("mgr").Dot("cache").BlockFunc(func(loop *jen.Group) {
			loop.Id("ids").Op("=").Append(jen.Id("ids"), jen.Id("key"))
		})
		group.Id("mgr").Dot("lock").Dot("RUnlock").Call()
		group.For().List(jen.Id("_"), jen.Id("id")).Op(":=").Range().Id("ids").BlockFunc(func(iter *jen.Group) {
			iter.List(jen.Id("_"), jen.Err()).Op(":=").Id("mgr").Dot("Update").Params(jen.Id("ctx"), jen.Id("id"))
			iter.If().Err().Op("!=").Nil().BlockFunc(func(fail *jen.Group) {
				fail.Return().Err()
			})
			iter.If(jen.Err().Op(":=").Id("ctx").Dot("Err").Call(), jen.Err().Op("!=").Nil()).BlockFunc(func(fail *jen.Group) {
				fail.Return().Err()
			})
		})
		group.Return().Nil()
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Set").Params(
		jen.Id("key").Add(cc.Key()),
		jen.Id("value").Add(cc.Value()),
	).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("FindOrCreate").Call(jen.Id("key")).Dot("Set").Call(jen.Id("value"))
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Purge").Params(jen.Id("key").Add(cc.Key())).BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("Lock").Call()
		if cc.WithEvents {
			group.List(jen.Id("old"), jen.Id("ok")).Op(":=").Id("mgr").Dot("cache").Index(jen.Id("key"))
			group.Delete(jen.Id("mgr").Dot("cache"), jen.Id("key"))
			group.Id("mgr").Dot("lock").Dot("Unlock").Call()
			group.If().Id("ok").Op("&&").Id("old").Dot("valid").BlockFunc(func(exists *jen.Group) {
				exists.Id("mgr").Dot("notify").Call(jen.Id("key"), jen.Id("old").Dot("data"), jen.Id(cc.EventTypeName("Delete")))
			})
		} else {
			group.Defer().Id("mgr").Dot("lock").Dot("Unlock").Call()
			group.Delete(jen.Id("mgr").Dot("cache"), jen.Id("key"))
		}
	}).Line()

	code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("PurgeAll").Params().BlockFunc(func(group *jen.Group) {
		group.Id("mgr").Dot("lock").Dot("Lock").Call()
		if cc.WithEvents {
			group.Id("mgr").Dot("cache").Op("=").Make(jen.Map(cc.Key()).Op("*").Id(cc.CacheType()))
			group.Id("mgr").Dot("lock").Dot("Unlock").Call()
			group.Var().Id("defKey").Add(cc.Key())
			group.Var().Id("defValue").Add(cc.Value())
			group.Id("mgr").Dot("notify").Call(jen.Id("defKey"), jen.Id("defValue"), jen.Id(cc.EventTypeName("DeleteAll")))
		} else {
			group.Id("mgr").Dot("cache").Op("=").Make(jen.Map(cc.Key()).Op("*").Id(cc.CacheType()))
			group.Defer().Id("mgr").Dot("lock").Dot("Unlock").Call()
		}

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
	if cc.WithEvents {
		code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("Subscribe").Params(
			jen.Id("handlerFunc").Id(cc.EventType()),
		).BlockFunc(func(group *jen.Group) {
			group.Id("mgr").Dot("listeners").Op("=").Append(jen.Id("mgr").Dot("listeners"), jen.Id("handlerFunc"))
		})
		code = code.Line().Func().Params(jen.Id("mgr").Op("*").Id(cc.TypeName)).Id("notify").Params(
			jen.Id("key").Add(cc.Key()),
			jen.Id("value").Add(cc.Value()),
			jen.Id("event").Id(cc.EventKindType()),
		).BlockFunc(func(group *jen.Group) {
			group.Id("n").Op(":=").Len(jen.Id("mgr").Dot("listeners"))
			group.For(jen.Id("i").Op(":=").Lit(0), jen.Id("i").Op("<").Id("n"), jen.Id("i").Op("++")).BlockFunc(func(iter *jen.Group) {
				iter.Id("mgr").Dot("listeners").Index(jen.Id("i")).Call(
					jen.Id("key"),
					jen.Id("value"),
					jen.Id("event"),
				)
			})
		})
	}
	return code
}

func (cc *CacheGen) generateCacheNode() jen.Code {
	code := jen.Type().Id(cc.CacheType()).StructFunc(func(group *jen.Group) {
		group.Id("valid").Bool()
		group.Id("lock").Qual("sync", "Mutex")
		group.Id("data").Add(cc.Value())
		group.Id("key").Add(cc.Key())
		group.Id("updater").Add(jen.Id(cc.UpdaterType()))
		if cc.WithEvents {
			group.Id("store").Op("*").Id(cc.TypeName)
		}
		if cc.WithExpiration {
			group.Id("updatedAt").Qual("time", "Time")
			group.Id("keepAlive").Qual("time", "Duration")
		}
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Valid").Params().Bool().BlockFunc(func(group *jen.Group) {
		var cond = jen.Empty()
		if cc.WithExpiration {
			cond = jen.Op("&&").Qual("time", "Now").Call().Dot("Sub").Call(jen.Id("cache").Dot("updatedAt")).Op("<").Id("cache").Dot("keepAlive")
		}
		group.Return().Id("cache").Dot("valid").Add(cond)
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

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Force").Params(jen.Id("ctx").Qual("context", "Context")).Params(cc.Value(), jen.Error()).BlockFunc(func(group *jen.Group) {
		group.Err().Op(":=").Id("cache").Dot("Update").Call(jen.Id("ctx"), jen.True())
		group.Return(jen.Id("cache").Dot("data"), jen.Err())
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Set").Params(jen.Id("value").Add(cc.Value())).BlockFunc(func(group *jen.Group) {
		group.Id("cache").Dot("lock").Dot("Lock").Call()
		group.Id("cache").Dot("data").Op("=").Id("value")
		group.Id("cache").Dot("valid").Op("=").True()
		if cc.WithExpiration {
			group.Id("cache").Dot("updatedAt").Op("=").Qual("time", "Now").Call()
		}
		group.Id("cache").Dot("lock").Dot("Unlock").Call()
		if cc.WithEvents {
			group.Id("cache").Dot("store").Dot("notify").Call(jen.Id("cache").Dot("key"), jen.Id("value"), jen.Id(cc.EventTypeName("Update")))
		}
	}).Line()

	code = code.Line().Func().Params(jen.Id("cache").Op("*").Id(cc.CacheType())).Id("Update").Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("force").Bool()).Error().BlockFunc(func(group *jen.Group) {
		group.Id("cache").Dot("lock").Dot("Lock").Call()
		var cond = jen.Empty()
		if cc.WithExpiration {
			cond = jen.Op("&&").Qual("time", "Now").Call().Dot("Sub").Call(jen.Id("cache").Dot("updatedAt")).Op("<").Id("cache").Dot("keepAlive")
		}

		group.If(jen.Id("cache").Dot("valid").Op("&&").Op("!").Id("force").Add(cond)).BlockFunc(func(ok *jen.Group) {
			ok.Id("cache").Dot("lock").Dot("Unlock").Call()
			ok.Return().Nil()
		})
		group.Id("temp").Op(",").Err().Op(":=").Id("cache").Dot("updater").Dot("Update").Call(
			jen.Id("ctx"), jen.Id("cache").Dot("key"))
		group.If(jen.Err().Op("!=").Nil()).BlockFunc(func(fail *jen.Group) {
			fail.Id("cache").Dot("lock").Dot("Unlock").Call()
			fail.Return().Err()
		})
		group.Id("cache").Dot("data").Op("=").Id("temp")
		group.Id("cache").Dot("valid").Op("=").True()
		if cc.WithExpiration {
			group.Id("cache").Dot("updatedAt").Op("=").Qual("time", "Now").Call()
		}
		group.Id("cache").Dot("lock").Dot("Unlock").Call()
		if cc.WithEvents {
			group.Id("cache").Dot("store").Dot("notify").Call(jen.Id("cache").Dot("key"), jen.Id("temp"), jen.Id(cc.EventTypeName("Update")))
		}
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
	if cc.WithEvents {
		code = code.Line().Type().Id(cc.EventType()).Func().Params(jen.Id("key").Add(cc.Key()), jen.Id("value").Add(cc.Value()), jen.Id("kind").Id(cc.EventKindType()))
		code = code.Line().Type().Id(cc.EventKindType()).Int()
		code = code.Line().Const().DefsFunc(func(group *jen.Group) {
			group.Id(cc.EventTypeName("Update")).Id(cc.EventKindType()).Op("=").Lit(0b01).Comment("0b01")
			group.Id(cc.EventTypeName("Delete")).Id(cc.EventKindType()).Op("=").Lit(0b10).Comment("0b10")
			group.Id(cc.EventTypeName("DeleteAll")).Id(cc.EventKindType()).Op("=").Lit(0b110).Comment("0b110")
		})
	}
	return code
}
