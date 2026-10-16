package api

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	DumpEvent   = "dump:progress"
	ImportEvent = "import:progress"
)

var dumpFilters = []runtime.FileFilter{{DisplayName: "SQL dump", Pattern: "*.sql;*.gz;*.dump;*.backup"}}

func (a *App) dumper(id string) (db.Dumper, error) {
	s, err := a.session(id)
	if err != nil {
		return nil, err
	}
	if d, ok := s.(db.Dumper); ok {
		return d, nil
	}
	return nil, db.ErrNotSupported
}

func (a *App) ExportDump(sessionID, jobID, name string, opts model.DumpOptions) (model.DumpResult, error) {
	d, err := a.dumper(sessionID)
	if err != nil {
		return model.DumpResult{}, err
	}
	path, err := runtime.SaveFileDialog(a.context(), runtime.SaveDialogOptions{
		Title:                "Export dump",
		DefaultFilename:      name,
		Filters:              dumpFilters,
		CanCreateDirectories: true,
	})
	if err != nil || path == "" {
		return model.DumpResult{}, err
	}
	if !strings.HasSuffix(strings.ToLower(path), ".sql") && !strings.HasSuffix(strings.ToLower(path), ".gz") {
		path += ".sql"
	}
	ctx, done := a.cancellable(jobID, 0)
	defer done()
	res := model.DumpResult{Path: path}
	err = writeDump(path, func(w io.Writer) error {
		return d.Dump(ctx, opts, w, func(p model.DumpProgress) {
			res.Bytes, res.Rows, res.Tables = p.Bytes, p.Rows, p.Done
			runtime.EventsEmit(a.ctx, DumpEvent, jobID, p)
		})
	})
	if ctx.Err() != nil {
		return res, ctx.Err()
	}
	return res, err
}

func writeDump(path string, dump func(io.Writer) error) error {
	part := path + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	buf := bufio.NewWriterSize(f, 1<<20)
	var w io.Writer = buf
	var zw *gzip.Writer
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		zw = gzip.NewWriter(buf)
		w = zw
	}
	err = dump(w)
	if err == nil && zw != nil {
		err = zw.Close()
	}
	if err == nil {
		err = buf.Flush()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(part, path)
	}
	if err != nil {
		os.Remove(part)
	}
	return err
}

func (a *App) PickDumpFile() (model.DumpFile, error) {
	path, err := runtime.OpenFileDialog(a.context(), runtime.OpenDialogOptions{Title: "Import dump", Filters: dumpFilters})
	if err != nil || path == "" {
		return model.DumpFile{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return model.DumpFile{}, err
	}
	return model.DumpFile{Path: path, Name: filepath.Base(path), Size: info.Size()}, nil
}

func (a *App) ImportDump(sessionID, jobID, path string, opts model.ImportOptions) (model.ImportResult, error) {
	d, err := a.dumper(sessionID)
	if err != nil {
		return model.ImportResult{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return model.ImportResult{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return model.ImportResult{}, err
	}
	counted := &countingReader{r: f}
	r, packed, err := dumpReader(counted)
	if err != nil {
		return model.ImportResult{}, err
	}
	ctx, done := a.cancellable(jobID, 0)
	defer done()
	return d.Import(ctx, r, info.Size(), opts, func(p model.ImportProgress) {
		if packed || p.Bytes < 0 {
			p.Bytes = counted.n
		}
		runtime.EventsEmit(a.ctx, ImportEvent, jobID, p)
	})
}

func dumpReader(f io.Reader) (io.Reader, bool, error) {
	br := bufio.NewReaderSize(f, 1<<16)
	head, _ := br.Peek(5)
	switch {
	case len(head) >= 2 && head[0] == 0x1f && head[1] == 0x8b:
		zr, err := gzip.NewReader(br)
		return zr, true, err
	}
	return br, false, nil
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
