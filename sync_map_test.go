package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

func TestSyncMapGen_Generate(t *testing.T) {
	st := SyncMapGen{
		TypeName:    "UserManager",
		KeyType:     "int64",
		ValueType:   "User",
		ValueImport: "example.com/xyz",
	}
	code := st.Generate()
	f := jen.NewFile("zzz")
	f.Add(code)
	err := f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
