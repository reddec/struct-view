package main

import (
	"github.com/dave/jennifer/jen"
	"github.com/jessevdk/go-flags"
	"os"
	"path/filepath"
	structview "struct-view"
)

type Config struct {
	SourceDir  string            `short:"d" long:"source-dir" env:"SOURCE_DIR" description:"Source directory" default:"."`
	SourceType string            `short:"f" long:"source-type" env:"SOURCE_TYPE" description:"Source struct type" required:"yes"`
	Package    string            `short:"p" long:"package" env:"PACKAGE" description:"Package name (can be override by output dir)" default:"mapping"`
	TargetDir  string            `short:"D" long:"target-dir" env:"TARGET_DIR" description:"Target directory"  default:"."`
	TargetType string            `short:"t" long:"target-type" env:"TARGET_TYPE" description:"Target struct type" required:"yes"`
	Func       string            `short:"F" long:"func" env:"FUNC" description:"Convert func name (if empty - To<TypeName>)"`
	Strict     bool              `long:"strict" env:"STRICT" description:"Require all fields be mapped"`
	Remap      map[string]string `short:"r" long:"remap" env:"REMAP" description:"Rename fields"`
	Output     string            `short:"o" long:"output" env:"OUTPUT" description:"Generated output destination (- means STDOUT)" default:"-"`
	Search     struct {
		Contains bool `long:"contains" env:"CONTAINS" description:"Try to find suitable fields just by part of field name"`
	} `group:"search option" namespace:"search" env-namespace:"SEARCH"`
	Args struct {
		Directory string `help:"override source and target type directory"`
	} `positional-args:"yes"`
}

func main() {
	var config Config
	_, err := flags.Parse(&config)
	if err != nil {
		os.Exit(1)
	}

	if config.Args.Directory != "" {
		config.SourceDir = config.Args.Directory
		config.TargetDir = config.Args.Directory
	}

	src, err := structview.LoadStruct(config.SourceDir, config.SourceType)
	if err != nil {
		panic(err)
	}

	dest, err := structview.LoadStruct(config.TargetDir, config.TargetType)
	if err != nil {
		panic(err)
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
	fnName := config.Func
	if fnName == "" {
		fnName = "To" + config.TargetType
	}

	cfg := structview.ToConvert{
		Source:         *src,
		Target:         *dest,
		FnName:         fnName,
		Remap:          config.Remap,
		SearchContains: config.Search.Contains,
	}
	mapping := cfg.Convert()
	if config.Strict && mapping.NotMatched != 0 {
		os.Exit(2)
	}
	out.Add(mapping.Code)
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
