package structview

import "github.com/dave/jennifer/jen"

type RingBuffer struct {
	Name         string
	TypeName     string
	Import       string
	Synchronized bool
}

func (rb *RingBuffer) Generate() jen.Code {
	out := jen.Empty()
	out.Add(rb.generateConstructor()).Line().Line()
	out.Add(rb.generateType()).Line().Line()
	out.Add(rb.generateMethods())
	return out
}

func (rb *RingBuffer) Qual() jen.Code {
	if rb.Import != "" {
		return jen.Qual(rb.Import, rb.TypeName)
	}
	return jen.Id(rb.TypeName)
}

func (rb *RingBuffer) fn() *jen.Statement {
	return jen.Func().Call(jen.Id("rb").Op("*").Id(rb.Name))
}

func (rb *RingBuffer) generateType() jen.Code {
	return jen.Comment("Ring buffer for type " + rb.TypeName).Line().Type().Id(rb.Name).StructFunc(func(group *jen.Group) {
		group.Id("seq").Uint64()
		group.Id("data").Index().Add(rb.Qual())
		if rb.Synchronized {
			group.Id("lock").Qual("sync", "RWMutex")
		}
	})
}

func (rb *RingBuffer) generateConstructor() jen.Code {
	return jen.Comment("New instance of ring buffer").Line().Func().Id("New" + rb.Name).Params(jen.Id("size").Uint()).Op("*").Id(rb.Name).BlockFunc(func(group *jen.Group) {
		group.Return().Op("&").Id(rb.Name).Values(jen.Id("data").Op(":").Make(jen.Index().Add(rb.Qual()), jen.Id("size")))
	}).Line().Line().Comment("Wrap pre-allocated buffer to ring buffer").Line().Func().Id("Wrap" + rb.Name).Params(jen.Id("buffer").Index().Add(rb.Qual())).Op("*").Id(rb.Name).BlockFunc(func(group *jen.Group) {
		group.Return().Op("&").Id(rb.Name).Values(jen.Id("data").Op(":").Id("buffer"))
	})
}

func (rb *RingBuffer) generateMethods() jen.Code {
	return jen.Comment("Add new element to the ring buffer. If buffer is full, the oldest element will be overwritten").Line().Add(rb.fn().Id("Add").Params(jen.Id("value").Add(rb.Qual())).BlockFunc(func(method *jen.Group) {
		if rb.Synchronized {
			method.Id("rb").Dot("lock").Dot("Lock").Call()
			method.Defer().Id("rb").Dot("lock").Dot("Unlock").Call()
		}
		slot := jen.Id("rb").Dot("seq").Op("%").Uint64().Call(jen.Len(jen.Id("rb").Dot("data")))
		method.Id("rb").Dot("data").Index(slot).Op("=").Id("value")
		method.Id("rb").Dot("seq").Op("++")
	})).Line().Line().Comment("Get element by index. Negative index is counting from end").Line().Add(rb.fn().Id("Get").Params(jen.Id("index").Int()).Params(jen.Id("ans").Add(rb.Qual())).BlockFunc(func(method *jen.Group) {
		if rb.Synchronized {
			method.Id("rb").Dot("lock").Dot("RLock").Call()
			method.Defer().Id("rb").Dot("lock").Dot("RUnlock").Call()
		}
		method.If().Id("index").Op("<").Lit(0).BlockFunc(func(group *jen.Group) {
			group.Id("index").Op("=").Len(jen.Id("rb").Dot("data")).Op("+").Id("index")
		})
		idx := jen.Parens(jen.Id("rb").Dot("seq").Op("+").Uint64().Call(jen.Id("index"))).Op("%").Uint64().Call(jen.Len(jen.Id("rb").Dot("data")))
		method.Return().Id("rb").Dot("data").Index(idx)
	})).Line().Line().Comment("Length of used elements. Always in range between zero and maximum capacity").Line().Add(rb.fn().Id("Len").Params().Int().BlockFunc(func(method *jen.Group) {
		if rb.Synchronized {
			method.Id("rb").Dot("lock").Dot("RLock").Call()
			method.Defer().Id("rb").Dot("lock").Dot("RUnlock").Call()
		}
		method.If(jen.Id("n").Op(":=").Len(jen.Id("rb").Dot("data")), jen.Id("rb").Dot("seq").Op("<").Uint64().Call(jen.Id("n"))).BlockFunc(func(early *jen.Group) {
			early.Return().Int().Call(jen.Id("rb").Dot("seq"))
		}).Else().BlockFunc(func(late *jen.Group) {
			late.Return().Id("n")
		})
	})).Line().Line().Comment("Clone of ring buffer with shallow copy of underlying buffer").Line().Add(rb.fn().Id("Clone").Params().Op("*").Id(rb.Name).BlockFunc(func(method *jen.Group) {
		if rb.Synchronized {
			method.Id("rb").Dot("lock").Dot("RLock").Call()
			method.Defer().Id("rb").Dot("lock").Dot("RUnlock").Call()
		}
		method.Id("cp").Op(":=").Make(jen.Index().Add(rb.Qual()), jen.Len(jen.Id("rb").Dot("data")))
		method.Copy(jen.Id("cp"), jen.Id("rb").Dot("data"))
		method.Return().Op("&").Id(rb.Name).ValuesFunc(func(vals *jen.Group) {
			vals.Id("seq").Op(":").Id("rb").Dot("seq")
			vals.Id("data").Op(":").Id("cp")
		})
	})).Line().Line().Comment("Flatten copy of underlying buffer. Data is always ordered in an insertion order").Line().Add(rb.fn().Id("Flatten").Params().Index().Add(rb.Qual()).BlockFunc(func(method *jen.Group) {
		if rb.Synchronized {
			method.Id("rb").Dot("lock").Dot("RLock").Call()
			method.Defer().Id("rb").Dot("lock").Dot("RUnlock").Call()
		}
		method.Id("ulen").Op(":=").Uint64().Call(jen.Len(jen.Id("rb").Dot("data")))
		method.Id("cp").Op(":=").Make(jen.Index().Add(rb.Qual()), jen.Id("ulen"))
		method.If(jen.Id("rb").Dot("seq").Op("<").Id("ulen")).BlockFunc(func(early *jen.Group) {
			early.Copy(jen.Id("cp"), jen.Id("rb").Dot("data"))
			early.Return().Id("cp").Index(jen.Empty(), jen.Id("rb").Dot("seq"))
		})
		method.Id("sep").Op(":=").Id("rb").Dot("seq").Op("%").Id("ulen")
		method.Id("head").Op(":=").Id("rb").Dot("data").Index(jen.Id("sep"), jen.Empty())

		method.Copy(jen.Id("cp"), jen.Id("head"))
		method.Copy(jen.Id("cp").Index(jen.Len(jen.Id("head")), jen.Empty()), jen.Id("rb").Dot("data").Index(jen.Empty(), jen.Id("sep")))
		method.Return().Id("cp")
	}))
}
