package crash

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	crashFile   = "crash.log"
	seenFile    = ".crash-seen"
	maxCrashLog = 2 << 20
)

func Dir(dataDir string) string { return filepath.Join(dataDir, "logs") }

func Watch(dir, header string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, crashFile)
	size := fileSize(path)
	seen, known := readSize(filepath.Join(dir, seenFile))
	crashed := known && size > seen
	if size > maxCrashLog && !crashed {
		_ = os.Rename(path, path+".old")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fmt.Fprintf(f, "\n--- %s started %s\n", header, time.Now().Format(time.RFC3339))
	if st, err := f.Stat(); err == nil {
		_ = os.WriteFile(filepath.Join(dir, seenFile), []byte(strconv.FormatInt(st.Size(), 10)), 0o600)
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		return "", err
	}
	if crashed {
		return path, nil
	}
	return "", nil
}

func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

func readSize(path string) (int64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	return n, err == nil
}
