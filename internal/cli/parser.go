package cli

import (
	"fmt"
	"strings"
)

type Parsed struct {
	Positionals []string
	Flags       map[string]string
}

func Parse(args []string) (Parsed, error) {
	parsed := Parsed{Flags: map[string]string{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			parsed.Positionals = append(parsed.Positionals, args[i+1:]...)
			return parsed, nil
		}
		if !strings.HasPrefix(arg, "--") {
			parsed.Positionals = append(parsed.Positionals, arg)
			continue
		}
		name, value, ok := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if name == "" {
			return Parsed{}, fmt.Errorf("invalid empty flag %q", arg)
		}
		if !ok {
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				i++
				value = args[i]
			} else {
				value = "true"
			}
		}
		parsed.Flags[name] = value
	}
	return parsed, nil
}
