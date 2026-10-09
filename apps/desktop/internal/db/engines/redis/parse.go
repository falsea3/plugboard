package redis

import (
	"fmt"
	"strconv"
	"strings"
)

type command struct {
	text string
	args []string
	line int
}

func parseScript(script string) ([]command, error) {
	var out []command
	for i, line := range strings.Split(script, "\n") {
		text := strings.TrimSpace(line)
		if text == "" || strings.HasPrefix(text, "#") || strings.HasPrefix(text, "//") {
			continue
		}
		args, err := splitArgs(text)
		if err != nil {
			return out, &lineError{line: i, text: text, err: err}
		}
		out = append(out, command{text: text, args: args, line: i})
	}
	return out, nil
}

type lineError struct {
	line int
	text string
	err  error
}

func (e *lineError) Error() string { return fmt.Sprintf("line %d: %v", e.line+1, e.err) }

func splitArgs(line string) ([]string, error) {
	var args []string
	i := 0
	for i < len(line) {
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= len(line) {
			break
		}
		var arg strings.Builder
		for i < len(line) && line[i] != ' ' && line[i] != '\t' {
			switch line[i] {
			case '"':
				n, err := readDouble(line[i:], &arg)
				if err != nil {
					return nil, err
				}
				i += n
			case '\'':
				n, err := readSingle(line[i:], &arg)
				if err != nil {
					return nil, err
				}
				i += n
			default:
				arg.WriteByte(line[i])
				i++
			}
		}
		args = append(args, arg.String())
	}
	return args, nil
}

func readSingle(s string, out *strings.Builder) (int, error) {
	for i := 1; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '\'':
			out.WriteByte('\'')
			i++
		case s[i] == '\'':
			return i + 1, nil
		default:
			out.WriteByte(s[i])
		}
	}
	return 0, fmt.Errorf("unbalanced quotes")
}

func readDouble(s string, out *strings.Builder) (int, error) {
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			return i + 1, nil
		}
		if c != '\\' || i+1 >= len(s) {
			out.WriteByte(c)
			continue
		}
		i++
		switch s[i] {
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case 'b':
			out.WriteByte('\b')
		case 'a':
			out.WriteByte('\a')
		case 'x':
			if i+2 < len(s) {
				if b, err := strconv.ParseUint(s[i+1:i+3], 16, 8); err == nil {
					out.WriteByte(byte(b))
					i += 2
					continue
				}
			}
			out.WriteByte('x')
		default:
			out.WriteByte(s[i])
		}
	}
	return 0, fmt.Errorf("unbalanced quotes")
}
