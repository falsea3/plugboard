package postgres

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

const (
	sectionNone = iota + 1
	sectionPreData
	sectionData
	sectionPostData
)

type dumpEntry struct {
	desc     string
	ns       string
	tag      string
	section  int
	defn     string
	copyStmt string
	rows     func(w io.Writer) error
	oids     []uint32
	blob     func(oid uint32, w io.Writer) error
}

func (e dumpEntry) hasData() bool { return e.rows != nil || e.blob != nil }

type dumpOut struct {
	w        io.Writer
	bytes    int64
	err      error
	progress func(model.DumpProgress)
	state    model.DumpProgress
	last     time.Time
	entries  []dumpEntry
	sec      int
}

func (o *dumpOut) Write(p []byte) (int, error) {
	if o.err != nil {
		return 0, o.err
	}
	n, err := o.w.Write(p)
	o.bytes += int64(n)
	o.err = err
	if o.progress != nil && time.Since(o.last) > 200*time.Millisecond {
		o.report()
	}
	return n, err
}

func (o *dumpOut) report() {
	o.last = time.Now()
	o.state.Bytes = o.bytes
	if o.progress != nil {
		o.progress(o.state)
	}
}

func (o *dumpOut) obj(desc, ns, tag, sql string) {
	if sql != "" {
		o.entries = append(o.entries, dumpEntry{desc: desc, ns: ns, tag: tag, section: o.sec, defn: sql + "\n"})
	}
}

func (o *dumpOut) line(sql string) { o.obj("", "", "", sql) }

func (o *dumpOut) data(e dumpEntry) {
	e.section = sectionData
	o.entries = append(o.entries, e)
}

type dumpHead struct {
	db      string
	server  string
	version string
	schemas []string
	at      time.Time
}

const sessionSettings = "SET statement_timeout = 0;\nSET lock_timeout = 0;\nSET idle_in_transaction_session_timeout = 0;\nSET client_encoding = 'UTF8';\nSET standard_conforming_strings = on;\nSELECT pg_catalog.set_config('search_path', '', false);\nSET check_function_bodies = false;\nSET xmloption = content;\nSET client_min_messages = warning;\nSET row_security = off;\n\n"

func writePlain(o *dumpOut, h dumpHead) error {
	fmt.Fprintf(o, "--\n-- Plugboard dump of database %s\n-- %s\n-- Schemas: %s\n-- %s\n--\n\n", h.db, h.server, strings.Join(h.schemas, ", "), h.at.Format(time.RFC3339))
	fmt.Fprint(o, sessionSettings)
	for _, e := range o.entries {
		if e.desc != "" {
			prefix := ""
			if e.hasData() {
				prefix = "Data for "
			}
			fmt.Fprintf(o, "--\n-- %sName: %s; Type: %s; Schema: %s\n--\n\n", prefix, oneLine(e.tag), e.desc, nonDash(e.ns))
		}
		fmt.Fprint(o, e.defn)
		if e.defn != "" {
			fmt.Fprint(o, "\n")
		}
		if err := plainData(o, e); err != nil {
			return err
		}
	}
	return o.err
}

func plainData(o *dumpOut, e dumpEntry) error {
	if e.rows != nil {
		fmt.Fprint(o, e.copyStmt)
		if err := e.rows(o); err != nil {
			return err
		}
		fmt.Fprint(o, "\n")
	}
	for _, oid := range e.oids {
		lw := &loWriter{w: o, oid: oid}
		if err := e.blob(oid, lw); err != nil {
			return err
		}
		if err := lw.flush(); err != nil {
			return err
		}
		fmt.Fprint(o, "\n")
	}
	return o.err
}

func oneLine(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "\r", " "), "\n", " ") }

func nonDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
