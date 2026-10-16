package postgres

import (
	"bufio"
	"compress/zlib"
	"fmt"
	"io"
)

type archiveOut struct {
	w   io.Writer
	err error
}

func (a *archiveOut) bytes(p []byte) {
	if a.err == nil {
		_, a.err = a.w.Write(p)
	}
}

func (a *archiveOut) byte(b byte) { a.bytes([]byte{b}) }

func (a *archiveOut) int(v int64) {
	sign := byte(0)
	if v < 0 {
		sign, v = 1, -v
	}
	a.bytes([]byte{sign, byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)})
}

func (a *archiveOut) str(s string) {
	a.int(int64(len(s)))
	a.bytes([]byte(s))
}

func (a *archiveOut) null() { a.int(-1) }

const formatCustom, formatTar = 1, 3

func archiveEntries(o *dumpOut) []dumpEntry {
	return append([]dumpEntry{
		{desc: "ENCODING", tag: "ENCODING", section: sectionPreData, defn: "SET client_encoding = 'UTF8';\n"},
		{desc: "STDSTRINGS", tag: "STDSTRINGS", section: sectionPreData, defn: "SET standard_conforming_strings = 'on';\n"},
		{desc: "SEARCHPATH", tag: "SEARCHPATH", section: sectionPreData, defn: "SELECT pg_catalog.set_config('search_path', '', false);\n"},
	}, o.entries...)
}

func (a *archiveOut) head(h dumpHead, format byte, level int64) {
	a.bytes([]byte("PGDMP"))
	a.bytes([]byte{1, 14, 0, 4, 8, format})
	a.int(level)
	t := h.at
	for _, v := range []int{t.Second(), t.Minute(), t.Hour(), t.Day(), int(t.Month()) - 1, t.Year() - 1900, 0} {
		a.int(int64(v))
	}
	a.str(h.db)
	a.str(h.version)
	a.str("Plugboard")
}

func (a *archiveOut) toc(entries []dumpEntry, extra func(id int, e dumpEntry)) {
	a.int(int64(len(entries)))
	for i, e := range entries {
		id := i + 1
		a.int(int64(id))
		hadData := int64(0)
		if e.hasData() {
			hadData = 1
		}
		a.int(hadData)
		a.str("0")
		a.str("0")
		a.str(e.tag)
		desc := e.desc
		if desc == "" {
			desc = "SQL"
		}
		a.str(desc)
		a.int(int64(e.section))
		a.str(e.defn)
		a.str("")
		a.str(e.copyStmt)
		a.str(e.ns)
		a.null()
		a.null()
		a.str("")
		a.str("false")
		if id > 1 {
			a.str(fmt.Sprint(id - 1))
		}
		a.null()
		extra(id, e)
	}
}

type chunkOut struct{ a *archiveOut }

func (c chunkOut) Write(p []byte) (int, error) {
	c.a.int(int64(len(p)))
	c.a.bytes(p)
	return len(p), c.a.err
}

func (a *archiveOut) packed(write func(io.Writer) error) error {
	buf := bufio.NewWriterSize(chunkOut{a}, 64<<10)
	z := zlib.NewWriter(buf)
	err := write(z)
	if cerr := z.Close(); err == nil {
		err = cerr
	}
	if ferr := buf.Flush(); err == nil {
		err = ferr
	}
	a.int(0)
	if err == nil {
		err = a.err
	}
	return err
}

func writeCustom(o *dumpOut, h dumpHead) error {
	a := &archiveOut{w: o}
	entries := archiveEntries(o)
	a.head(h, formatCustom, -1)
	a.toc(entries, func(_ int, e dumpEntry) {
		flag := byte(offsetNoData)
		if e.hasData() {
			flag = 1
		}
		a.byte(flag)
		a.bytes(make([]byte, 8))
	})
	for i, e := range entries {
		switch {
		case e.rows != nil:
			a.byte(blockData)
			a.int(int64(i + 1))
			if err := a.packed(e.rows); err != nil {
				return err
			}
		case e.blob != nil:
			a.byte(blockBlobs)
			a.int(int64(i + 1))
			for _, oid := range e.oids {
				a.int(int64(int32(oid)))
				if err := a.packed(func(w io.Writer) error { return e.blob(oid, w) }); err != nil {
					return err
				}
			}
			a.int(0)
		}
	}
	if a.err != nil {
		return a.err
	}
	return o.err
}
