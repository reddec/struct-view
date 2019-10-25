package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

func TestTimedCache_Generate(t *testing.T) {
	st := &TimedCache{
		TypeName:    "CacheMyClass",
		ValueType:   "*MyCalss",
		ValueImport: "github.com/example/xyz",
		Array:       true,
	}
	code := st.Generate()
	f := jen.NewFile("zzz")
	f.Add(code)
	err := f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
