package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"vanguard/internal/evaluator"
	"vanguard/internal/parser"
	"vanguard/internal/reader"
	"vanguard/internal/reporter"
)

// run keeps CLI behavior testable without terminating the test process.
func run(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("vanguard", flag.ContinueOnError)
	fs.SetOutput(errOut)
	policyPath := fs.String("p", "policy.yaml", "YAML policy path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(errOut, "usage: vanguard [-p policy.yaml] report.json|- ")
		return 2
	}
	policy, err := evaluator.LoadPolicy(*policyPath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	input, err := reader.Open(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	if input != os.Stdin {
		defer input.Close()
	}
	report, err := parser.Parse(input)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	verdict := evaluator.Evaluate(report, policy)
	reporter.PrintTable(verdict, policy, out)
	if !verdict.Passed {
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
