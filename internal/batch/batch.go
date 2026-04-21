package batch

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"sync"
)

type Command struct {
	Line int
	Raw  string
	Args []string
}

type Result struct {
	Command Command
	Err     error
}

type Executor func(context.Context, int, Command) error

type Options struct {
	Workers  int
	FailFast bool
	Execute  Executor
}

func ExpandCSV(r io.Reader, template string) ([]Command, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("csv did not contain any rows")
	}
	headers := map[string]int{}
	for i, cell := range rows[0] {
		key := strings.TrimSpace(cell)
		if key != "" {
			headers[key] = i
		}
	}
	commands := []Command{}
	for rowNum, row := range rows[1:] {
		values := map[string]string{}
		for header, index := range headers {
			if index < len(row) {
				values[header] = strings.TrimSpace(row[index])
			}
		}
		rendered, err := renderTemplate(template, values)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum+2, err)
		}
		args, err := splitArgs(rendered)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum+2, err)
		}
		if len(args) == 0 {
			continue
		}
		if args[0] == "gws" {
			args = args[1:]
		}
		if len(args) == 0 {
			continue
		}
		commands = append(commands, Command{
			Line: rowNum + 2,
			Raw:  rendered,
			Args: args,
		})
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("csv did not expand into any commands")
	}
	return commands, nil
}

func Parse(r io.Reader) ([]Command, error) {
	scanner := bufio.NewScanner(r)
	commands := []Command{}
	for lineNum := 1; scanner.Scan(); lineNum++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		args, err := splitArgs(raw)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		if len(args) == 0 {
			continue
		}
		if args[0] == "gws" {
			args = args[1:]
		}
		if len(args) == 0 {
			continue
		}
		commands = append(commands, Command{
			Line: lineNum,
			Raw:  raw,
			Args: args,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read batch file: %w", err)
	}
	return commands, nil
}

func Run(ctx context.Context, commands []Command, opts Options) []Result {
	if len(commands) == 0 {
		return nil
	}
	workers := opts.Workers
	if workers <= 0 {
		workers = 1
	}
	if opts.Execute == nil {
		results := make([]Result, len(commands))
		for i, command := range commands {
			results[i] = Result{Command: command, Err: fmt.Errorf("batch executor is not configured")}
		}
		return results
	}
	type job struct {
		index   int
		command Command
	}
	results := make([]Result, len(commands))
	jobs := make(chan job)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				err := opts.Execute(ctx, item.index, item.command)
				results[item.index] = Result{Command: item.command, Err: err}
				if err != nil && opts.FailFast {
					cancel()
				}
			}
		}()
	}
	for index, command := range commands {
		if opts.FailFast && ctx.Err() != nil {
			for i := index; i < len(commands); i++ {
				results[i] = Result{
					Command: commands[i],
					Err:     context.Canceled,
				}
			}
			close(jobs)
			wg.Wait()
			return results
		}
		select {
		case <-ctx.Done():
			if opts.FailFast {
				for i := index; i < len(commands); i++ {
					results[i] = Result{
						Command: commands[i],
						Err:     context.Canceled,
					}
				}
				close(jobs)
				wg.Wait()
				return results
			}
		case jobs <- job{index: index, command: command}:
		}
	}
	close(jobs)
	wg.Wait()
	return results
}

func splitArgs(line string) ([]string, error) {
	args := []string{}
	var current strings.Builder
	inQuote := byte(0)
	escaped := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}
		switch ch {
		case '\\':
			escaped = true
		case '\'', '"':
			if inQuote == 0 {
				inQuote = ch
				continue
			}
			if inQuote == ch {
				inQuote = 0
				continue
			}
			current.WriteByte(ch)
		case ' ', '\t':
			if inQuote != 0 {
				current.WriteByte(ch)
				continue
			}
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if escaped {
		current.WriteByte('\\')
	}
	if inQuote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args, nil
}

func renderTemplate(template string, values map[string]string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(template); {
		if i+1 < len(template) && template[i] == '{' && template[i+1] == '{' {
			end := strings.Index(template[i+2:], "}}")
			if end < 0 {
				return "", fmt.Errorf("unterminated template placeholder")
			}
			key := strings.TrimSpace(template[i+2 : i+2+end])
			value, ok := values[key]
			if !ok {
				return "", fmt.Errorf("missing csv column %q", key)
			}
			out.WriteString(value)
			i += 2 + end + 2
			continue
		}
		out.WriteByte(template[i])
		i++
	}
	return out.String(), nil
}
