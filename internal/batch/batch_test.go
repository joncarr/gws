package batch

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParseSkipsCommentsAndUnderstandsQuotes(t *testing.T) {
	commands, err := Parse(strings.NewReader(`
# comment
gws update group eng@example.com --name "Engineering Team"
print users --query 'isSuspended=false'
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("len(commands) = %d", len(commands))
	}
	if got := strings.Join(commands[0].Args, "|"); got != "update|group|eng@example.com|--name|Engineering Team" {
		t.Fatalf("first args = %q", got)
	}
	if got := strings.Join(commands[1].Args, "|"); got != "print|users|--query|isSuspended=false" {
		t.Fatalf("second args = %q", got)
	}
}

func TestRunUsesWorkerPool(t *testing.T) {
	commands := []Command{
		{Line: 1, Args: []string{"version"}},
		{Line: 2, Args: []string{"version"}},
		{Line: 3, Args: []string{"version"}},
		{Line: 4, Args: []string{"version"}},
	}
	var mu sync.Mutex
	running := 0
	maxRunning := 0
	results := Run(context.Background(), commands, Options{
		Workers: 2,
		Execute: func(ctx context.Context, index int, command Command) error {
			mu.Lock()
			running++
			if running > maxRunning {
				maxRunning = running
			}
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			running--
			mu.Unlock()
			return nil
		},
	})
	if len(results) != 4 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if maxRunning < 2 {
		t.Fatalf("maxRunning = %d, want at least 2", maxRunning)
	}
}

func TestRunFailFastMarksLaterCommandsCanceled(t *testing.T) {
	commands := []Command{
		{Line: 1, Args: []string{"one"}},
		{Line: 2, Args: []string{"two"}},
		{Line: 3, Args: []string{"three"}},
	}
	results := Run(context.Background(), commands, Options{
		Workers:  1,
		FailFast: true,
		Execute: func(ctx context.Context, index int, command Command) error {
			if command.Line == 1 {
				return context.DeadlineExceeded
			}
			return nil
		},
	})
	if results[0].Err == nil {
		t.Fatal("first result error = nil")
	}
	if results[1].Err != context.Canceled {
		t.Fatalf("second result error = %v", results[1].Err)
	}
	if results[2].Err != context.Canceled {
		t.Fatalf("third result error = %v", results[2].Err)
	}
}

func TestExpandCSV(t *testing.T) {
	commands, err := ExpandCSV(strings.NewReader("email,orgUnit\nada@example.com,/Engineering\n"), `update user "{{email}}" --org-unit "{{orgUnit}}"`)
	if err != nil {
		t.Fatalf("ExpandCSV() error = %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("len(commands) = %d", len(commands))
	}
	if got := strings.Join(commands[0].Args, "|"); got != "update|user|ada@example.com|--org-unit|/Engineering" {
		t.Fatalf("args = %q", got)
	}
	if commands[0].Line != 2 {
		t.Fatalf("line = %d", commands[0].Line)
	}
}

func TestExpandCSVMissingColumn(t *testing.T) {
	_, err := ExpandCSV(strings.NewReader("email\nada@example.com\n"), `update user "{{missing}}"`)
	if err == nil {
		t.Fatal("ExpandCSV() error = nil")
	}
	if !strings.Contains(err.Error(), `missing csv column "missing"`) {
		t.Fatalf("error = %v", err)
	}
}
