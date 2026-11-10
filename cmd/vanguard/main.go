package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/theprideofwolves/vanguard/internal/gate"
	"github.com/theprideofwolves/vanguard/internal/reader"
	"io"
	"net/http"
	"os"
	"time"
)

func run(args []string, out, errOut io.Writer) int {
	if len(args) > 0 && args[0] == "compare" {
		return runCompare(args[1:], out, errOut)
	}
	fs := flag.NewFlagSet("vanguard", flag.ContinueOnError)
	fs.SetOutput(errOut)
	policyPath := fs.String("p", "policy.yaml", "YAML policy path")
	asJSON := fs.Bool("json", false, "Machine-readable decision on stdout")
	notify := fs.Bool("notify", false, "On failure notify Slack using VANGUARD_SLACK_WEBHOOK")
	fs.Usage = func() {
		fmt.Fprintln(errOut, "Usage: vanguard [flags] report.json|-\nNo arguments: local UI.\nPolicy impact: vanguard compare -h\nEvaluation exits: 0 PASS, 1 policy FAIL, 2 error.")
		fs.PrintDefaults()
	}
	if e := fs.Parse(args); e != nil {
		if e == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(errOut, "usage: vanguard [-p policy.yaml] [-json] [-notify] report.json|-")
		return 2
	}
	input, e := reader.Open(fs.Arg(0))
	if e != nil {
		fmt.Fprintln(errOut, e)
		return 2
	}
	if input != os.Stdin {
		defer input.Close()
	}
	data, e := io.ReadAll(io.LimitReader(input, (16<<20)+1))
	if e != nil {
		fmt.Fprintln(errOut, e)
		return 2
	}
	p, e := os.Open(*policyPath)
	if e != nil {
		fmt.Fprintln(errOut, e)
		return 2
	}
	defer p.Close()
	policy, e := io.ReadAll(io.LimitReader(p, (1<<20)+1))
	if e != nil {
		fmt.Fprintln(errOut, e)
		return 2
	}
	d, e := gate.Evaluate(data, policy)
	if e != nil {
		fmt.Fprintln(errOut, e)
		return 2
	}
	if *asJSON {
		e = json.NewEncoder(out).Encode(d)
	} else {
		for _, r := range d.Rows {
			limit := "unlimited"
			if r.Limited {
				limit = fmt.Sprint(r.Allowed)
			}
			if _, err := fmt.Fprintf(out, "%s: observed=%d allowed=%s exceeded=%t\n", r.Severity, r.Observed, limit, r.Exceeded); err != nil {
				fmt.Fprintln(errOut, "cannot write decision")
				return 2
			}
		}
		result := "PASS"
		if !d.Passed {
			result = "FAIL"
		}
		_, e = fmt.Fprintf(out, "Result: %s\nPolicy SHA256: %s\n", result, d.PolicyHash)
	}
	if e != nil {
		fmt.Fprintln(errOut, "cannot write decision")
		return 2
	}
	if *notify && !d.Passed {
		client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		if e = gate.Notify(context.Background(), client, os.Getenv("VANGUARD_SLACK_WEBHOOK"), d, os.Getenv("VANGUARD_RUN_URL")); e != nil {
			fmt.Fprintln(errOut, e)
		}
	}
	if !d.Passed {
		return 1
	}
	return 0
}
func main() {
	if len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "-ui") {
		if e := launchUI(); e != nil {
			fmt.Println(e)
			fmt.Println("Press Enter to close")
			fmt.Scanln()
			os.Exit(2)
		}
		return
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
