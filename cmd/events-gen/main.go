package main

import (
	"github.com/dave/jennifer/jen"
	"github.com/jessevdk/go-flags"
	structview "github.com/reddec/struct-view"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Package    string            `short:"p" long:"package" env:"PACKAGE" description:"Package name (can be override by output dir)" default:"events"`
	Output     string            `short:"o" long:"output" env:"OUTPUT" description:"Generated output destination (- means STDOUT)" default:"-"`
	Private    bool              `short:"P" long:"private" env:"PRIVATE" description:"Make generated event structures be private by prefix 'event'"`
	EventBus   string            `long:"event-bus" env:"EVENT_BUS" description:"Generate structure that aggregates all events" default:""`
	Mirror     bool              `short:"m" long:"mirror" env:"MIRROR" description:"Mirror all events to the universal emitter"`
	FromMirror bool              `short:"f" long:"from-mirror" env:"FROM_MIRROR" description:"Create producer events as from mirror (only for event bus)"`
	IgnoreCase bool              `short:"i" long:"ignore-case" env:"IGNORE_CASE" description:"Ignore event case for universal source (--from-mirror)"`
	Sink       bool              `short:"s" long:"sink" env:"SINK" description:"Make a sink method for event bus to subscribe to all events"`
	Hint       map[string]string `short:"H" long:"hint" env:"HINT" description:"Give a hint about events (eventName -> struct name)"`
	Args       struct {
		Directories []string `help:"source directories (by default - current)"`
	} `positional-args:"yes"`
}

func main() {
	var config Config
	_, err := flags.Parse(&config)
	if err != nil {
		os.Exit(1)
	}

	if len(config.Args.Directories) == 0 {
		config.Args.Directories = []string{"."}
	}

	var out *jen.File
	if config.Output != "-" {
		pkg, err := structview.FindPackage(filepath.Dir(config.Output))
		if err != nil {
			// fallback
			out = jen.NewFile(config.Package)
		} else {
			out = jen.NewFilePathName(pkg, filepath.Base(pkg))
		}
	} else {
		out = jen.NewFile(config.Package)
	}
	ev := structview.EventGenerator{
		WithMirror: config.Mirror,
		WithBus:    config.EventBus != "",
		WithSink:   config.Sink,
		BusName:    config.EventBus,
		Private:    config.Private,
		Hints:      config.Hint,
	}
	code, err := ev.Generate(config.Args.Directories...)
	if err != nil {
		log.Fatal(err)
	}
	out.Add(code)
	var output = os.Stdout
	if config.Output != "-" {
		output, err = os.Create(config.Output)
		if err != nil {
			panic(err)
		}
		defer output.Close()
	}
	err = out.Render(output)
	if err != nil {
		panic(err)
	}
}
