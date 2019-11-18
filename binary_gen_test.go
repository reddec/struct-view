package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

func TestBinaryGenerator_Generate(t *testing.T) {
	st := BinaryGenerator{
		TypeName: "User",
	}
	code, pack, err := st.Generate("examples/binarygen")
	if err != nil {
		t.Error(err)
		return
	}
	if pack != "binarygen" {
		t.Error("no package?")
	}
	f := jen.NewFile(pack)
	f.Add(code)
	err = f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
