package parser

import (
	"strings"
	"testing"
)

func TestReportEdgeCases(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		valid       bool
	}{
		{"invalid UTF8", "{\"target\":\"\xff\",\"findings\":[]}", false},
		{"empty", "", false},
		{"trailing garbage", `{"findings":[]}x`, false},
		{"escaped duplicate key", `{"findings":[],"\u0066indings":[]}`, false},
		{"nested duplicate", `{"findings":[{"severity":"high","id":"a","ID":"b"}]}`, false},
		{"wrong array type", `{"findings":{}}`, false},
		{"wrong field type", `{"findings":[{"severity":"high","references":[2]}]}`, false},
		{"deep input", strings.Repeat("[", 10001) + strings.Repeat("]", 10001), false},
		{"same ID distinct finding", `{"findings":[{"id":"a","severity":"high","target":"x"}]}`, false},
		{"distinct descriptions", `{"findings":[{"id":"a","severity":"high","description":"one"},{"id":"a","severity":"high","description":"two"}]}`, true},
		{"reordered identical", `{"findings":[{"id":"a","severity":"high"},{"severity":"HIGH","id":"a"}]}`, false},
		{"Unicode", `{"target":"日本語","findings":[{"severity":"INFO","description":"Résumé"}]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.input))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}

func TestReportSizeBoundary(t *testing.T) {
	base := `{"findings":[]}`
	input := base + strings.Repeat(" ", (16<<20)-len(base))
	if _, err := Parse(strings.NewReader(input)); err != nil {
		t.Fatal("exact limit rejected", err)
	}
	if _, err := Parse(strings.NewReader(input + " ")); err == nil {
		t.Fatal("over limit accepted")
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{`{"findings":[]}`, `null`, `{"findings":[{"severity":"high"}]}`, `{"findings":[],"Findings":[]}`, "[", "\xff"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 1<<20 {
			t.Skip()
		}
		r, err := Parse(strings.NewReader(input))
		if err == nil && (r == nil || r.Findings == nil) {
			t.Fatal("accepted report without findings array")
		}
	})
}
