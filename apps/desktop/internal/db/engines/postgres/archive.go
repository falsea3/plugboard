package postgres

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const (
	blockData    = 1
	blockBlobs   = 3
	offsetNoData = 3
)

func archiveVersion(major, minor, rev byte) int { return int(major)<<16 | int(minor)<<8 | int(rev) }

var (
	v1_2  = archiveVersion(1, 2, 0)
	v1_3  = archiveVersion(1, 3, 0)
	v1_4  = archiveVersion(1, 4, 0)
	v1_5  = archiveVersion(1, 5, 0)
	v1_6  = archiveVersion(1, 6, 0)
	v1_7  = archiveVersion(1, 7, 0)
	v1_8  = archiveVersion(1, 8, 0)
	v1_9  = archiveVersion(1, 9, 0)
	v1_10 = archiveVersion(1, 10, 0)
	v1_11 = archiveVersion(1, 11, 0)
	v1_12 = archiveVersion(1, 12, 0)
	v1_14 = archiveVersion(1, 14, 0)
	v1_15 = archiveVersion(1, 15, 0)
	v1_16 = archiveVersion(1, 16, 0)
)

type archive struct {
	r           *bufio.Reader
	err         error
	version     int
	intSize     int
	offSize     int
	compression byte
	entries     []tocEntry
}

type optStr struct {
	s  string
	ok bool
}

type tocEntry struct {
	id         int
	desc       string
	tag        string
	defn       string
	dropStmt   string
	copyStmt   string
	namespace  string
	tablespace optStr
	tableAM    optStr
	relkind    int
	owner      string
	deps       []int
	hasData    bool
}

var errNotArchive = errors.New("not a pg_dump archive")

func (a *archive) byte() byte {
	if a.err != nil {
		return 0
	}
	b, err := a.r.ReadByte()
	a.fail(err)
	return b
}

func (a *archive) fail(err error) {
	if err != nil && a.err == nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		a.err = fmt.Errorf("the pg_dump archive is cut short or damaged: %w", err)
	}
}

func (a *archive) int() int64 {
	sign := byte(0)
	if a.version > archiveVersion(1, 0, 0) {
		sign = a.byte()
	}
	var v int64
	for i := range a.intSize {
		v |= int64(a.byte()) << (8 * i)
	}
	if sign != 0 {
		v = -v
	}
	return v
}

func (a *archive) str() optStr {
	n := a.int()
	if n < 0 || a.err != nil {
		return optStr{}
	}
	if n > 1<<30 {
		a.fail(fmt.Errorf("a %d-byte string", n))
		return optStr{}
	}
	buf := make([]byte, n)
	_, err := io.ReadFull(a.r, buf)
	a.fail(err)
	return optStr{string(buf), err == nil}
}

func (a *archive) offset() int {
	if a.version < v1_7 {
		switch i := a.int(); {
		case i < 0:
			return 1
		case i == 0:
			return offsetNoData
		}
		return 2
	}
	flag := a.byte()
	for range a.offSize {
		a.byte()
	}
	return int(flag)
}

func readArchive(r *bufio.Reader) (*archive, error) {
	a := &archive{r: r}
	magic := make([]byte, 5)
	if _, err := io.ReadFull(r, magic); err != nil || string(magic) != "PGDMP" {
		return nil, errNotArchive
	}
	major, minor := a.byte(), a.byte()
	rev := byte(0)
	if major > 1 || major == 1 && minor > 0 {
		rev = a.byte()
	}
	a.version = archiveVersion(major, minor, rev)
	if major != 1 || a.version > archiveVersion(1, 16, 255) {
		return nil, fmt.Errorf("pg_dump archive version %d.%d is newer than this app reads", major, minor)
	}
	a.intSize = int(a.byte())
	a.offSize = a.intSize
	if a.version >= v1_7 {
		a.offSize = int(a.byte())
	}
	if format := a.byte(); format != 1 {
		return nil, fmt.Errorf("this pg_dump archive is in format %d; only the custom format (-Fc) is read here", format)
	}
	a.header()
	a.toc()
	if a.err != nil {
		return nil, a.err
	}
	return a, nil
}

func (a *archive) header() {
	switch {
	case a.version >= v1_15:
		a.compression = a.byte()
	case a.version >= v1_2:
		level := int64(0)
		if a.version < v1_4 {
			level = int64(a.byte())
		} else {
			level = a.int()
		}
		if level != 0 {
			a.compression = 1
		}
	default:
		a.compression = 1
	}
	if a.version >= v1_4 {
		for range 7 {
			a.int()
		}
		a.str()
	}
	if a.version >= v1_10 {
		a.str()
		a.str()
	}
}

func (a *archive) toc() {
	n := a.int()
	for i := int64(0); i < n && a.err == nil; i++ {
		var e tocEntry
		e.id = int(a.int())
		a.int()
		if a.version >= v1_8 {
			a.str()
		}
		a.str()
		e.tag = a.str().s
		e.desc = a.str().s
		if a.version >= v1_11 {
			a.int()
		}
		e.defn = a.str().s
		e.dropStmt = a.str().s
		if a.version >= v1_3 {
			e.copyStmt = a.str().s
		}
		if a.version >= v1_6 {
			e.namespace = a.str().s
		}
		if a.version >= v1_10 {
			e.tablespace = a.str()
		}
		if a.version >= v1_14 {
			e.tableAM = a.str()
		}
		if a.version >= v1_16 {
			e.relkind = int(a.int())
		}
		e.owner = a.str().s
		if a.version >= v1_9 {
			a.str()
		}
		if a.version >= v1_5 {
			for a.err == nil {
				d := a.str()
				if !d.ok {
					break
				}
				id, _ := strconv.Atoi(d.s)
				e.deps = append(e.deps, id)
			}
		}
		e.hasData = a.offset() != offsetNoData
		if a.version < v1_7 {
			a.int()
		}
		a.entries = append(a.entries, e)
	}
}
