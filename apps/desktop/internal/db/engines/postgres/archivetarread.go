package postgres

import (
	"archive/tar"
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type tarSource struct {
	tr   *tar.Reader
	seen map[string]bool
}

func readTarArchive(r io.Reader) (*archive, error) {
	tr := tar.NewReader(r)
	h, err := tr.Next()
	if err != nil {
		return nil, fmt.Errorf("the tar archive can't be read: %w", err)
	}
	if h.Name != "toc.dat" {
		return nil, fmt.Errorf("this tar file isn't a pg_dump archive: it starts with %s, not toc.dat", h.Name)
	}
	a, err := readArchive(bufio.NewReader(tr))
	if err != nil {
		return nil, err
	}
	if a.format != formatTar {
		return nil, errors.New("toc.dat in this tar file doesn't describe a tar archive")
	}
	a.tar = &tarSource{tr: tr, seen: map[string]bool{}}
	return a, nil
}

func (t *tarSource) next() (*tar.Header, error) {
	h, err := t.tr.Next()
	if err == io.EOF {
		return nil, io.ErrUnexpectedEOF
	}
	if h != nil {
		t.seen[h.Name] = true
	}
	return h, err
}

func (t *tarSource) open(name string) error {
	if t.seen[name] {
		return fmt.Errorf("%s comes earlier in the tar archive than its place in the restore order", name)
	}
	for {
		h, err := t.next()
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if h.Name == name {
			return nil
		}
	}
}

func (t *tarSource) data(a *archive, e tocEntry, w io.Writer) error {
	if e.desc == "BLOBS" {
		return t.blobs(a, w)
	}
	if err := t.open(e.file); err != nil {
		return err
	}
	if e.copyStmt != "" {
		fmt.Fprint(w, e.copyStmt)
	}
	if _, err := io.Copy(w, t.tr); err != nil {
		return fmt.Errorf("data of %s: %w", e.tag, err)
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}

func (t *tarSource) blobs(a *archive, w io.Writer) error {
	found := false
	for {
		h, err := t.next()
		if err != nil {
			return err
		}
		oid, isBlob := blobFile(h.Name)
		if !isBlob {
			if found {
				return nil
			}
			continue
		}
		found = true
		if a.version < v1_12 {
			fmt.Fprintf(w, "SELECT pg_catalog.lo_create('%d');\n", oid)
		}
		lw := &loWriter{w: w, oid: oid}
		if _, err := io.Copy(lw, t.tr); err != nil {
			return err
		}
		if err := lw.flush(); err != nil {
			return err
		}
	}
}

func blobFile(name string) (uint32, bool) {
	rest, ok := strings.CutPrefix(name, "blob_")
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimSuffix(rest, ".dat"), 10, 32)
	return uint32(n), err == nil && n != 0
}
