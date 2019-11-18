package structview

import (
	"github.com/dave/jennifer/jen"
	"os"
	"testing"
)

func TestBBoltStorage_Render(t *testing.T) {
	bbs := BBoltStorage{Name: "User", TypeName: "UserRegistry", Compressed: true}
	out, err := bbs.Render("./examples/bbol")
	if err != nil {
		t.Error(err)
		return
	}

	f := jen.NewFile("xyz")
	f.Add(out)
	err = f.Render(os.Stdout)
	if err != nil {
		t.Error(err)
	}
}
