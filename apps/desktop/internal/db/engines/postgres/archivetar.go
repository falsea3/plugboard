package postgres

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

func tarFile(tw *tar.Writer, name string, at time.Time, write func(io.Writer) error) error {
	tmp, err := os.CreateTemp("", "plugboard-dump-*.dat")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	buf := bufio.NewWriterSize(tmp, 1<<20)
	if err := write(buf); err != nil {
		return err
	}
	if err := buf.Flush(); err != nil {
		return err
	}
	size, err := tmp.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := tw.WriteHeader(tarHeader(name, size, at)); err != nil {
		return err
	}
	_, err = io.Copy(tw, tmp)
	return err
}

func tarHeader(name string, size int64, at time.Time) *tar.Header {
	return &tar.Header{Name: name, Mode: 0o600, Size: size, ModTime: at, Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}
}

func writeTar(o *dumpOut, h dumpHead) error {
	tw := tar.NewWriter(o)
	entries := archiveEntries(o)
	var toc bytes.Buffer
	a := &archiveOut{w: &toc}
	a.head(h, formatTar, 0)
	a.toc(entries, func(id int, e dumpEntry) {
		name := ""
		if e.hasData() {
			name = fmt.Sprintf("%d.dat", id)
		}
		a.str(name)
	})
	if err := tw.WriteHeader(tarHeader("toc.dat", int64(toc.Len()), h.at)); err != nil {
		return err
	}
	if _, err := tw.Write(toc.Bytes()); err != nil {
		return err
	}
	for i, e := range entries {
		id := i + 1
		if e.rows != nil {
			if err := tarFile(tw, fmt.Sprintf("%d.dat", id), h.at, e.rows); err != nil {
				return err
			}
		}
		if e.blob == nil {
			continue
		}
		var list bytes.Buffer
		for _, oid := range e.oids {
			name := fmt.Sprintf("blob_%d.dat", oid)
			fmt.Fprintf(&list, "%d %s\n", oid, name)
			if err := tarFile(tw, name, h.at, func(w io.Writer) error { return e.blob(oid, w) }); err != nil {
				return err
			}
		}
		if err := tw.WriteHeader(tarHeader(fmt.Sprintf("blobs_%d.toc", id), int64(list.Len()), h.at)); err != nil {
			return err
		}
		if _, err := tw.Write(list.Bytes()); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return o.err
}
