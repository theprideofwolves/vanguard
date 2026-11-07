package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONAndNotificationExit(t *testing.T) {
	t.Setenv("VANGUARD_SLACK_WEBHOOK", "")
	var out, errs bytes.Buffer
	code := run([]string{"-p", "../../policy.yaml", "-json", "-notify", "../../testdata/report-fail.json"}, &out, &errs)
	if code != 1 || !strings.Contains(out.String(), "PolicyHash") || !strings.Contains(errs.String(), "invalid Slack") {
		t.Fatal(code, out.String(), errs.String())
	}
}
