package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriterTableAlignsColumns(t *testing.T) {
	var out bytes.Buffer
	err := New(&out).Table([][]string{
		{"Domain", "Primary", "Verified", "Aliases"},
		{"example.com", "yes", "verified", "0"},
		{"long.example.com", "no", "pending", "10"},
	})
	if err != nil {
		t.Fatalf("Table() error = %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), out.String())
	}

	columnStarts := []struct {
		header string
		values []string
	}{
		{header: "Primary", values: []string{"yes", "no"}},
		{header: "Verified", values: []string{"verified", "pending"}},
		{header: "Aliases", values: []string{"0", "10"}},
	}

	for _, column := range columnStarts {
		want := strings.Index(lines[0], column.header)
		if want < 0 {
			t.Fatalf("header %q not found in %q", column.header, lines[0])
		}
		for i, value := range column.values {
			got := strings.Index(lines[i+1], value)
			if got != want {
				t.Fatalf("column %q row %d starts at %d, want %d:\n%s", column.header, i+1, got, want, out.String())
			}
		}
	}
}

func TestWriterCSV(t *testing.T) {
	var out bytes.Buffer
	err := New(&out).CSV([][]string{
		{"Header One", "Header Two"},
		{"value", "value,with,comma"},
	})
	if err != nil {
		t.Fatalf("CSV() error = %v", err)
	}
	if out.String() != "Header One,Header Two\nvalue,\"value,with,comma\"\n" {
		t.Fatalf("output = %q", out.String())
	}
}
