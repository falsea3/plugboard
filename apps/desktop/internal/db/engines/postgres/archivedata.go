package postgres

import (
	"compress/zlib"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

type chunks struct {
	a    *archive
	left int64
	done bool
}

func (c *chunks) Read(p []byte) (int, error) {
	for c.left == 0 && !c.done {
		n := c.a.int()
		if c.a.err != nil {
			return 0, c.a.err
		}
		if n <= 0 {
			c.done = true
		}
		c.left = max(n, 0)
	}
	if c.done {
		return 0, io.EOF
	}
	if int64(len(p)) > c.left {
		p = p[:c.left]
	}
	n, err := c.a.r.Read(p)
	c.left -= int64(n)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return n, err
}

func (a *archive) unpack(w io.Writer) error {
	c := &chunks{a: a}
	var r io.Reader = c
	switch a.compression {
	case 0:
	case 1:
		z, err := zlib.NewReader(c)
		if err != nil {
			return err
		}
		defer z.Close()
		r = z
	case 2:
		r = lz4.NewReader(c)
	case 3:
		z, err := zstd.NewReader(c)
		if err != nil {
			return err
		}
		defer z.Close()
		r = z
	default:
		return fmt.Errorf("the pg_dump archive uses compression %d, which this app doesn't read", a.compression)
	}
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	_, err := io.Copy(io.Discard, c)
	return err
}

func (a *archive) skipChunks() error {
	_, err := io.Copy(io.Discard, &chunks{a: a})
	return err
}

func (a *archive) blockHeader() (typ, id int, eof bool) {
	b, err := a.r.ReadByte()
	if errors.Is(err, io.EOF) {
		return 0, 0, true
	}
	a.fail(err)
	if a.version < v1_3 {
		a.r.UnreadByte()
		b = blockData
	}
	return int(b), int(a.int()), false
}

func (a *archive) findBlock(id int, skipped map[int]bool) (int, error) {
	if skipped[id] {
		return 0, fmt.Errorf("the data of entry %d comes earlier in the archive than its place in the restore order", id)
	}
	for {
		typ, got, eof := a.blockHeader()
		if a.err != nil {
			return 0, a.err
		}
		if eof {
			return 0, fmt.Errorf("the data of entry %d is missing from the archive", id)
		}
		if got == id {
			return typ, nil
		}
		skipped[got] = true
		var err error
		if typ == blockBlobs {
			err = a.blobs(nil)
		} else {
			err = a.skipChunks()
		}
		if err != nil {
			return 0, err
		}
	}
}

const loChunk = 256 << 10

type loWriter struct {
	w   io.Writer
	oid uint32
	off int64
	buf []byte
}

func (l *loWriter) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 {
		take := min(len(p), loChunk-len(l.buf))
		l.buf = append(l.buf, p[:take]...)
		p = p[take:]
		if len(l.buf) == loChunk {
			if err := l.flush(); err != nil {
				return 0, err
			}
		}
	}
	return n, nil
}

func (l *loWriter) flush() error {
	if len(l.buf) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(l.w, "SELECT pg_catalog.lo_put('%d', %d, pg_catalog.decode('%s', 'hex'));\n", l.oid, l.off, hex.EncodeToString(l.buf))
	l.off += int64(len(l.buf))
	l.buf = l.buf[:0]
	return err
}

func (a *archive) blobs(w io.Writer) error {
	for {
		oid := uint32(int32(a.int()))
		if a.err != nil {
			return a.err
		}
		if oid == 0 {
			return nil
		}
		if w == nil {
			if err := a.skipChunks(); err != nil {
				return err
			}
			continue
		}
		if a.version < v1_12 {
			if _, err := fmt.Fprintf(w, "SELECT pg_catalog.lo_create('%d');\n", oid); err != nil {
				return err
			}
		}
		lw := &loWriter{w: w, oid: oid}
		if err := a.unpack(lw); err != nil {
			return err
		}
		if err := lw.flush(); err != nil {
			return err
		}
	}
}
