package postgres

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

func (d Dialect) ReadArchive(r io.Reader) (io.Reader, bool, error) {
	br := bufio.NewReaderSize(r, 1<<20)
	head, _ := br.Peek(512)
	var a *archive
	var err error
	switch {
	case len(head) >= 5 && string(head[:5]) == "PGDMP":
		a, err = readArchive(br)
	case len(head) >= 262 && string(head[257:262]) == "ustar":
		a, err = readTarArchive(br)
	default:
		return br, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	pr, pw := io.Pipe()
	go func() {
		w := bufio.NewWriterSize(pw, 1<<20)
		err := d.archiveSQL(a, w)
		if err == nil {
			err = w.Flush()
		}
		pw.CloseWithError(err)
	}()
	return pr, true, nil
}

const (
	passMain = iota
	passACL
	passPostACL
)

func restorePass(e tocEntry) int {
	switch e.desc {
	case "ACL", "ACL LANGUAGE", "DEFAULT ACL":
		return passACL
	case "EVENT TRIGGER", "MATERIALIZED VIEW DATA":
		return passPostACL
	case "COMMENT", "SECURITY LABEL":
		if strings.HasPrefix(e.tag, "EVENT TRIGGER ") {
			return passPostACL
		}
	}
	return passMain
}

func skipEntry(e tocEntry) bool {
	switch e.desc {
	case "ENCODING", "STDSTRINGS", "SEARCHPATH", "DATABASE", "DATABASE PROPERTIES":
		return true
	case "ACL", "COMMENT", "SECURITY LABEL":
		return strings.HasPrefix(e.tag, "DATABASE ")
	}
	return false
}

type archiveState struct {
	w          io.Writer
	byID       map[int]tocEntry
	skipped    map[int]bool
	searchPath bool
	schema     string
	tablespace optStr
	tableAM    optStr
}

func (d Dialect) archiveSQL(a *archive, w io.Writer) error {
	st := &archiveState{w: w, byID: map[int]tocEntry{}, skipped: map[int]bool{}}
	fmt.Fprint(w, "SET statement_timeout = 0;\nSET lock_timeout = 0;\nSET idle_in_transaction_session_timeout = 0;\n")
	for _, e := range a.entries {
		st.byID[e.id] = e
		switch e.desc {
		case "ENCODING", "STDSTRINGS", "SEARCHPATH":
			fmt.Fprintln(w, strings.TrimSpace(e.defn))
			st.searchPath = st.searchPath || e.desc == "SEARCHPATH"
		}
	}
	fmt.Fprint(w, "SET check_function_bodies = false;\nSET xmloption = content;\nSET client_min_messages = warning;\nSET row_security = off;\n\n")
	for pass := passMain; pass <= passPostACL; pass++ {
		for _, e := range a.entries {
			if restorePass(e) != pass || skipEntry(e) {
				continue
			}
			if err := d.archiveEntry(a, st, e); err != nil {
				return err
			}
		}
	}
	return a.err
}

func (d Dialect) archiveEntry(a *archive, st *archiveState, e tocEntry) error {
	w := st.w
	st.state(d, e)
	switch {
	case e.desc == "BLOB METADATA":
		perBlob(w, e.defn, "SELECT pg_catalog.lo_create('", "')")
	case e.desc == "ACL" && strings.HasPrefix(e.tag, "LARGE OBJECTS") && len(e.deps) > 0:
		blobACL(w, e.defn, st.byID[e.deps[0]].defn)
	case strings.TrimSpace(e.defn) != "":
		fmt.Fprintf(w, "%s\n\n", e.defn)
	}
	if e.owner != "" && e.dropStmt != "" {
		if e.desc == "BLOB METADATA" {
			perBlob(w, e.defn, "ALTER LARGE OBJECT ", " OWNER TO "+d.QuoteIdent(e.owner))
		} else if what := d.ownedObject(e); what != "" {
			fmt.Fprintf(w, "ALTER %s OWNER TO %s;\n\n", what, d.QuoteIdent(e.owner))
		}
	}
	if e.relkind == 'p' && e.tableAM.ok && e.tableAM.s != "" {
		fmt.Fprintf(w, "ALTER TABLE %s SET ACCESS METHOD %s;\n\n", d.qualified(e.namespace, e.tag), d.QuoteIdent(e.tableAM.s))
	}
	if !e.hasData || (e.desc != "TABLE DATA" && e.desc != "BLOBS") {
		return nil
	}
	if a.tar != nil {
		return a.tar.data(a, e, w)
	}
	typ, err := a.findBlock(e.id, st.skipped)
	if err != nil {
		return err
	}
	if typ == blockBlobs {
		return a.blobs(w)
	}
	if e.copyStmt != "" {
		fmt.Fprint(w, e.copyStmt)
	}
	if err := a.unpack(w); err != nil {
		return fmt.Errorf("data of %s: %w", e.tag, err)
	}
	_, err = fmt.Fprint(w, "\n")
	return err
}

func (st *archiveState) state(d Dialect, e tocEntry) {
	if !st.searchPath && e.namespace != "" && e.namespace != st.schema {
		st.schema = e.namespace
		path := d.QuoteIdent(e.namespace)
		if e.namespace != "pg_catalog" {
			path += ", pg_catalog"
		}
		fmt.Fprintf(st.w, "SET search_path = %s;\n\n", path)
	}
	if e.tablespace.ok && e.tablespace != st.tablespace {
		st.tablespace = e.tablespace
		value := "''"
		if e.tablespace.s != "" {
			value = d.QuoteIdent(e.tablespace.s)
		}
		fmt.Fprintf(st.w, "SET default_tablespace = %s;\n\n", value)
	}
	if e.relkind != 'p' && e.tableAM.ok && e.tableAM != st.tableAM {
		st.tableAM = e.tableAM
		fmt.Fprintf(st.w, "SET default_table_access_method = %s;\n\n", d.QuoteIdent(e.tableAM.s))
	}
}

func (d Dialect) qualified(namespace, name string) string {
	if namespace == "" {
		return d.QuoteIdent(name)
	}
	return d.QuoteIdent(namespace) + "." + d.QuoteIdent(name)
}

var ownedByName = map[string]bool{
	"COLLATION": true, "CONVERSION": true, "DOMAIN": true, "FOREIGN TABLE": true, "MATERIALIZED VIEW": true,
	"SEQUENCE": true, "STATISTICS": true, "TABLE": true, "TEXT SEARCH DICTIONARY": true, "TEXT SEARCH CONFIGURATION": true,
	"TYPE": true, "VIEW": true, "DATABASE": true, "PROCEDURAL LANGUAGE": true, "SCHEMA": true, "EVENT TRIGGER": true,
	"FOREIGN DATA WRAPPER": true, "SERVER": true, "PUBLICATION": true, "SUBSCRIPTION": true,
}

var ownedByDrop = map[string]bool{"AGGREGATE": true, "FUNCTION": true, "OPERATOR": true, "OPERATOR CLASS": true, "OPERATOR FAMILY": true, "PROCEDURE": true}

func (d Dialect) ownedObject(e tocEntry) string {
	switch {
	case ownedByName[e.desc]:
		return e.desc + " " + d.qualified(e.namespace, e.tag)
	case e.desc == "BLOB":
		return "LARGE OBJECT " + e.tag
	case ownedByDrop[e.desc] && strings.HasPrefix(e.dropStmt, "DROP "):
		return strings.TrimRight(e.dropStmt[5:], ";\n")
	}
	return ""
}

func perBlob(w io.Writer, oids, before, after string) {
	for _, oid := range strings.Fields(oids) {
		fmt.Fprintf(w, "%s%s%s;\n", before, oid, after)
	}
	fmt.Fprintln(w)
}

var largeObject = regexp.MustCompile(`LARGE OBJECT \d+`)

func blobACL(w io.Writer, acl, oids string) {
	for _, oid := range strings.Fields(oids) {
		fmt.Fprintln(w, largeObject.ReplaceAllLiteralString(acl, "LARGE OBJECT "+oid))
	}
	fmt.Fprintln(w)
}
