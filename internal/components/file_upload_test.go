package components

import (
	"encoding/json"
	"testing"
)

func TestFileUpload_RequiredPropsOnly(t *testing.T) {
	c := FileUpload("import-file", FileUploadProps{
		Name:        "file",
		Label:       "File",
		Placeholder: "Drop a file here or click to browse",
	})

	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	wantSubstrings := []string{
		`"type":"file_upload"`,
		`"id":"import-file"`,
		`"name":"file"`,
		`"label":"File"`,
		`"placeholder":"Drop a file here or click to browse"`,
	}
	for _, w := range wantSubstrings {
		if !containsStr(got, w) {
			t.Errorf("expected %q in JSON, got: %s", w, got)
		}
	}
	for _, omitted := range []string{`"hint"`, `"accept"`, `"max_size_bytes"`, `"prefill_filename"`} {
		if containsStr(got, omitted) {
			t.Errorf("expected %q to be omitted, got: %s", omitted, got)
		}
	}
}

func TestFileUpload_FullProps(t *testing.T) {
	c := FileUpload("import-file", FileUploadProps{
		Name:               "file",
		Label:              "File",
		Placeholder:        "Drop or browse",
		Hint:               "CSV up to 5 MB",
		Accept:             ".csv,.tsv",
		MaxSizeBytes:       5242880,
		ErrorMessageSize:   "Too large: {limit}",
		ErrorMessageFormat: "Unsupported.",
		PrefillFilename:    "old.csv",
		ReattachHint:       "Re-select the file to retry",
	})

	b, _ := json.Marshal(c)
	got := string(b)
	for _, w := range []string{
		`"hint":"CSV up to 5 MB"`,
		`"accept":".csv,.tsv"`,
		`"max_size_bytes":5242880`,
		`"error_message_size":"Too large: {limit}"`,
		`"error_message_format":"Unsupported."`,
		`"prefill_filename":"old.csv"`,
		`"reattach_hint":"Re-select the file to retry"`,
	} {
		if !containsStr(got, w) {
			t.Errorf("expected %q in JSON, got: %s", w, got)
		}
	}
}
