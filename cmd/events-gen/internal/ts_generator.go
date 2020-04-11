package internal

import (
	"bytes"
	"github.com/Masterminds/sprig"
	structview "github.com/reddec/struct-view"
	"github.com/reddec/struct-view/deepparser"
	"strings"
	"text/template"
)

//go:generate go-bindata -pkg internal ts.gotemplate
func GenerateTS(result *structview.EventGeneratorResult) string {
	var tsg deepparser.TypeScript
	fm := sprig.TxtFuncMap()
	fm["firstLine"] = func(text string) string {
		return strings.Split(text, "\n")[0]
	}
	fm["typescript"] = tsg.MapField
	fm["definitions"] = func() []*deepparser.Definition {
		if tsg.Ordered == nil {
			for _, event := range result.Events {
				tsg.AddFromDir(event.TypeName, event.Dir)
			}
		}
		return tsg.Ordered
	}
	t := template.Must(template.New("").Funcs(fm).Parse(string(MustAsset("ts.gotemplate"))))
	buffer := &bytes.Buffer{}
	err := t.Execute(buffer, result)
	if err != nil {
		panic(err)
	}
	return buffer.String()
}
