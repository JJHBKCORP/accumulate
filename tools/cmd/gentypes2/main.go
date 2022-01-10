package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/AccumulateNetwork/accumulate/tools/internal/typegen"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var flags struct {
	Package string
	Out     string
	IsState bool
}

func main() {
	cmd := cobra.Command{
		Use:  "gentypes [file]",
		Args: cobra.ExactArgs(1),
		Run:  run,
	}

	cmd.Flags().StringVar(&flags.Package, "package", "protocol", "Package name")
	cmd.Flags().StringVarP(&flags.Out, "out", "o", "types_gen.go", "Output file")

	_ = cmd.Execute()
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		fatalf("%v", err)
	}
}

func checkf(err error, format string, otherArgs ...interface{}) {
	if err != nil {
		fatalf(format+": %v", append(otherArgs, err)...)
	}
}

func readTypes(file string) map[string]typegen.Type {
	f, err := os.Open(file)
	check(err)
	defer f.Close()

	var types map[string]typegen.Type

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	err = dec.Decode(&types)
	check(err)

	return types
}

func run(_ *cobra.Command, args []string) {
	types := readTypes(args[0])
	ttypes := convert(types, flags.Package)

	w := new(bytes.Buffer)
	check(Go.Execute(w, ttypes))
	check(typegen.GoFmt(flags.Out, w))
}