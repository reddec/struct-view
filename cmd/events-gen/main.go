package main

import (
	"github.com/dave/jennifer/jen"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"path/filepath"
	structview "struct-view"
)

type Config struct {
	Package  string `short:"p" long:"package" env:"PACKAGE" description:"Package name (can be override by output dir)" default:"events"`
	Output   string `short:"o" long:"output" env:"OUTPUT" description:"Generated output destination (- means STDOUT)" default:"-"`
	Private  bool   `short:"P" long:"private" env:"PRIVATE" description:"Make generated event structures be private by prefix 'event'"`
	EventBus string `long:"event-bus" env:"EVENT_BUS" description:"Generate structure that aggregates all events" default:""`
	Args     struct {
		Directory string `help:"source directory"`
	} `positional-args:"yes"`
}

func main() {
	var config Config
	_, err := flags.Parse(&config)
	if err != nil {
		os.Exit(1)
	}

	if config.Args.Directory == "" {
		config.Args.Directory = "."
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
		WithBus: config.EventBus != "",
		BusName: config.EventBus,
		Private: config.Private,
	}
	code, err := ev.Generate(config.Args.Directory)
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
