package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	config := generatorConfig{
		PackageName:   "logplugins",
		RegistrarName: "Registrar",
	}
	var appenders bindingFlag
	var layouts bindingFlag
	var filters bindingFlag
	var lookups bindingFlag
	var resolvers bindingFlag
	var outputPath string

	flags := flag.NewFlagSet("goark-log-plugin-gen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&config.PackageName, "package", config.PackageName, "generated Go package name")
	flags.StringVar(&config.RegistrarName, "registrar", config.RegistrarName, "registrar function name")
	flags.StringVar(&outputPath, "out", "", "output file path; stdout when empty")
	flags.Var(&appenders, "appender", "appender binding in kind=factory form")
	flags.Var(&layouts, "layout", "layout binding in kind=factory form")
	flags.Var(&filters, "filter", "filter binding in kind=factory form")
	flags.Var(&lookups, "lookup", "lookup binding in namespace=factory form")
	flags.Var(&resolvers, "json-template-resolver", "JSON Template resolver binding in kind=factory form")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	config.Appenders = appenders.Values()
	config.Layouts = layouts.Values()
	config.Filters = filters.Values()
	config.Lookups = lookups.Values()
	config.JSONTemplateResolvers = resolvers.Values()
	data, err := generatePluginRegistrar(config)
	if err != nil {
		fmt.Fprintf(stderr, "goark-log-plugin-gen: %v\n", err)
		return 1
	}
	if outputPath == "" {
		if _, err := stdout.Write(data); err != nil {
			fmt.Fprintf(stderr, "goark-log-plugin-gen: write stdout: %v\n", err)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fmt.Fprintf(stderr, "goark-log-plugin-gen: write %s: %v\n", outputPath, err)
		return 1
	}
	return 0
}
