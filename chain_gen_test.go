package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

func TestChainGen_Generate(t *testing.T) {
	eg := ChainGen{
		ContextType: "Message",
		Import:      "",
		TypeName:    "Pipe",
	}
	code := eg.Generate()
	f := jen.NewFile("xyz")
	f.Add(code)
	err := f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
