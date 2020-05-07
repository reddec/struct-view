package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

func TestCacheGen_Generate(t *testing.T) {
	st := CacheGen{
		TypeName:       "UserManager",
		KeyType:        "int64",
		ValueType:      "User",
		ValueImport:    "example.com/xyz",
		WithExpiration: true,
	}
	code := st.Generate()
	f := jen.NewFile("zzz")
	f.Add(code)
	err := f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
