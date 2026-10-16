package sqltext

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

const firstChunk = 1 << 20

type Scanner struct {
	r     io.Reader
	syn   Syntax
	buf   string
	pos   int
	delim string
	eof   bool
	read  int64
	chunk int
}

func NewScanner(r io.Reader, syn Syntax) *Scanner {
	return &Scanner{r: r, syn: syn, delim: ";", chunk: firstChunk}
}

func (sc *Scanner) Consumed() int64 {
	return sc.read - int64(len(sc.buf)-sc.pos)
}

func (sc *Scanner) Next() (string, error) {
	for {
		from, end, next, d, found := scanStatement(sc.buf, sc.pos, sc.delim, sc.syn, sc.eof)
		if found || sc.eof {
			sc.delim, sc.pos, sc.chunk = d, next, firstChunk
			stmt := strings.TrimSpace(sc.buf[from:end])
			if stmt != "" && (stmt[0] == '\\' || FirstKeyword(stmt) != "") {
				return stmt, nil
			}
			if !found {
				return "", io.EOF
			}
			continue
		}
		if err := sc.fill(); err != nil {
			return "", err
		}
	}
}

func (sc *Scanner) CopyData(w io.Writer) error {
	for {
		if nl := strings.IndexByte(sc.buf[sc.pos:], '\n'); nl >= 0 {
			sc.pos += nl + 1
			break
		}
		if sc.eof {
			sc.pos = len(sc.buf)
			return io.ErrUnexpectedEOF
		}
		if err := sc.fill(); err != nil {
			return err
		}
	}
	for {
		start := sc.pos
		for {
			nl := strings.IndexByte(sc.buf[sc.pos:], '\n')
			if nl < 0 {
				break
			}
			line := strings.TrimSuffix(sc.buf[sc.pos:sc.pos+nl], "\r")
			if line == `\.` {
				if _, err := io.WriteString(w, sc.buf[start:sc.pos]); err != nil {
					return err
				}
				sc.pos += nl + 1
				return nil
			}
			sc.pos += nl + 1
		}
		if _, err := io.WriteString(w, sc.buf[start:sc.pos]); err != nil {
			return err
		}
		if sc.eof {
			if rest := strings.TrimSpace(sc.buf[sc.pos:]); rest == `\.` || rest == "" {
				sc.pos = len(sc.buf)
				return nil
			}
			return io.ErrUnexpectedEOF
		}
		if err := sc.fill(); err != nil {
			return err
		}
	}
}

func (sc *Scanner) fill() error {
	if sc.eof {
		return io.EOF
	}
	rest := sc.buf[sc.pos:]
	var b bytes.Buffer
	b.Grow(len(rest) + sc.chunk)
	b.WriteString(rest)
	n, err := io.CopyN(&b, sc.r, int64(sc.chunk))
	sc.read += n
	if errors.Is(err, io.EOF) {
		sc.eof = true
	} else if err != nil {
		return err
	}
	sc.buf, sc.pos = b.String(), 0
	if sc.chunk < 64<<20 {
		sc.chunk *= 2
	}
	return nil
}
