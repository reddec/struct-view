package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

func TestEnumGen_Generate2(t *testing.T) {
	st := EnumGen{
		TargetType: "string",
		Name:       "Teams",
		Values:     []string{"Alpha", "Beta", "Gamma"},
	}
	code := st.Generate()
	f := jen.NewFile("main")
	f.Add(code)
	err := f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
