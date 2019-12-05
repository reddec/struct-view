package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

// event:"xxxx"
// event:"yyy"
// something wrong event:"zzz"
type event struct {
}

func TestEventGenerator_Generate(t *testing.T) {
	eg := EventGenerator{
		BusName:        "Events",
		WithMirror:     true,
		WithBus:        true,
		FromMirror:     true,
		FromIgnoreCase: true,
		PrivateEmit:    true,
	}
	code, err := eg.Generate(".")
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
