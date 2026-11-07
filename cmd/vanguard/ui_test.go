package main

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUIEvaluationAndExport(t *testing.T) {
	h := newUI("token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/", nil))
	if !strings.Contains(w.Body.String(), "background:#0b1627") {
		t.Fatal("CSS not rendered")
	}
	var b bytes.Buffer
	f := multipart.NewWriter(&b)
	f.WriteField("policy", samplePolicy)
	f.WriteField("report", sampleReport)
	f.Close()
	r := httptest.NewRequest("POST", "http://localhost/", &b)
	r.Header.Set("Content-Type", f.FormDataContentType())
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "FAIL") {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/export", nil))
	if !strings.Contains(w.Body.String(), "PolicyHash") {
		t.Fatal(w.Body.String())
	}
}
