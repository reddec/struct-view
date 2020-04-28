package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

func TestRingBuffer_Generate(t *testing.T) {
	tt := RingBuffer{
		Name:     "IntBuffer",
		TypeName: "int",
	}
	code := tt.Generate()
	f := jen.NewFilePathName("xyz", "AAAAAAA")
	f.Add(code)
	err := f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
