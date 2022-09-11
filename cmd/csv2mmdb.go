// This is a utility for converting CSV files to MaxMind's mmdb format.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fholzer/csv2mmdb/pkg/convert"
)

func main() {
	input := flag.String("input", "", "Path to the CSV input file (REQUIRED)")
	output := flag.String("output", "", "Path to the mmdb output file (REQUIRED)")
	configFilePath := flag.String("config", "", "Path to the configuration file (REQUIRED)")

	flag.Parse()

	var errors []string

	if *input == "" {
		errors = append(errors, "-block-file is required")
	}

	if *output == "" {
		errors = append(errors, "-output-file is required")
	}

	if *input != "" && *output != "" && *output == *input {
		errors = append(errors, "Your output file must be different than your block file(input file).")
	}

	args := flag.Args()
	if len(args) > 0 {
		errors = append(errors, "unknown argument(s): "+strings.Join(args, ", "))
	}

	if len(errors) != 0 {
		printHelp(errors)
		os.Exit(1)
	}

	config, err := convert.NewConfig(*configFilePath)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	err = convert.ConvertFile(config, *input, *output)
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp(errors []string) {
	var passedFlags []string
	flag.Visit(func(f *flag.Flag) {
		passedFlags = append(passedFlags, "-"+f.Name)
	})

	if len(passedFlags) > 0 {
		errors = append(errors, "flags passed: "+strings.Join(passedFlags, ", "))
	}

	for _, message := range errors {
		fmt.Fprintln(flag.CommandLine.Output(), message)
	}

	flag.Usage()
}
