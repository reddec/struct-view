package structview

import (
	"os"
	"testing"

	"github.com/dave/jennifer/jen"
)

func TestRingBuffer_Generate(t *testing.T) {
	tt := RingBuffer{
		Name:     "IntBuffer",
		TypeName: "int",
		Notify:   true,
	}
	code := tt.Generate()
	f := jen.NewFilePathName("xyz", "AAAAAAA")
	f.Add(code)
	err := f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
