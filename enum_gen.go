package structview

import (
	"github.com/dave/jennifer/jen"
	"strings"
)

type EnumGen struct {
	TargetType string
	Name       string
	Values     []string
}

func (eg EnumGen) Generate() jen.Code {
	out := jen.Empty()
	out.Add(eg.generateType()).Line()
	out.Add(eg.generateSourceGetter()).Line()
	out.Add(eg.generateValidator()).Line()
	out.Add(eg.generateJSONUnmarshal()).Line()
	return out
}

func (eg EnumGen) generateType() jen.Code {
	return jen.Type().Id(eg.Name).Id(eg.TargetType)
}

func (eg EnumGen) generateSourceGetter() jen.Code {
	return jen.Func().Params(jen.Id("v").Id(eg.Name)).Id("Get").Params().Id(eg.TargetType).BlockFunc(func(group *jen.Group) {
		group.Return().Id(eg.TargetType).Call(jen.Id("v"))
	})
}

func (eg EnumGen) generateValidator() jen.Code {
	isString := eg.TargetType == "string"
	return jen.Func().Params(jen.Id("v").Id(eg.Name)).Id("IsValid").Params().Bool().BlockFunc(func(group *jen.Group) {

		group.Switch(jen.Id(eg.TargetType).Call(jen.Id("v"))).BlockFunc(func(options *jen.Group) {
			var opts []jen.Code
			for _, v := range eg.Values {
				if isString {
					opts = append(opts, jen.Lit(v))
				} else {
					opts = append(opts, jen.Id(v))
				}
			}
			options.Case(opts...).BlockFunc(func(option *jen.Group) {
				option.Return(jen.True())
			})
			options.Default().Return(jen.False())
		})
	})
}

func (eg EnumGen) generateJSONUnmarshal() jen.Code {
	return jen.Func().Params(jen.Id("v").Op("*").Id(eg.Name)).Id("UnmarshalJSON").Params(jen.Id("data").Index().Byte()).Error().BlockFunc(func(group *jen.Group) {
		group.Var().Id("parsed").Id(eg.TargetType)
		group.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("parsed"))
		group.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Err()))
		group.Id("typed").Op(":=").Id(eg.Name).Call(jen.Id("parsed"))
		group.If(jen.Op("!").Id("typed").Dot("IsValid").Call()).BlockFunc(func(failed *jen.Group) {
			var possibleOpts = strings.Join(eg.Values, ", ")
			failed.Return().Qual("errors", "New").Call(jen.Lit("Invalid value for type " + eg.Name + ". Possible options are: " + possibleOpts))
		})
		group.Op("*").Id("v").Op("=").Id("typed")
		group.Return().Nil()
	})
}
