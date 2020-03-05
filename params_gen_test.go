package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

func TestParamsGen_Generate(t *testing.T) {
	tt := ParamsGen{
		Dir:        "examples/params",
		StructName: "Basic",
	}
	code, err := tt.Generate()
	if err != nil {
		t.Error(err)
		return
	}
	f := jen.NewFile("xyz")
	f.Add(code)
	err = f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
