package cli

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		positionals []string
		flags       map[string]string
	}{
		{
			name:        "positionals",
			args:        []string{"check", "connection"},
			positionals: []string{"check", "connection"},
			flags:       map[string]string{},
		},
		{
			name:        "flags",
			args:        []string{"setup", "--profile", "prod", "--domain=example.com", "--yes"},
			positionals: []string{"setup"},
			flags:       map[string]string{"profile": "prod", "domain": "example.com", "yes": "true"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(got.Positionals) != len(tt.positionals) {
				t.Fatalf("positionals = %v, want %v", got.Positionals, tt.positionals)
			}
			for i := range got.Positionals {
				if got.Positionals[i] != tt.positionals[i] {
					t.Fatalf("positionals = %v, want %v", got.Positionals, tt.positionals)
				}
			}
			for k, want := range tt.flags {
				if got.Flags[k] != want {
					t.Fatalf("flag %s = %q, want %q", k, got.Flags[k], want)
				}
			}
		})
	}
}
