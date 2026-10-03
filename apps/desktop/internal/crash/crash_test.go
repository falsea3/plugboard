package crash

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWatchTellsWhenTheLastRunCrashed(t *testing.T) {
	dir := t.TempDir()
	if path, err := Watch(dir, "Plugboard test"); err != nil || path != "" {
		t.Fatalf("first launch = %q, %v; want no crash", path, err)
	}
	if path, _ := Watch(dir, "Plugboard test"); path != "" {
		t.Fatalf("a clean relaunch reported a crash at %s", path)
	}
	f, err := os.OpenFile(filepath.Join(dir, crashFile), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("panic: boom\n\ngoroutine 1 [running]:\n")
	f.Close()
	path, err := Watch(dir, "Plugboard test")
	if err != nil || path != filepath.Join(dir, crashFile) {
		t.Fatalf("after a crash = %q, %v", path, err)
	}
	if path, _ := Watch(dir, "Plugboard test"); path != "" {
		t.Fatal("the same crash was reported twice")
	}
}

func TestAPanicInAnyGoroutineLandsInTheLog(t *testing.T) {
	if dir := os.Getenv("CRASH_TEST_DIR"); dir != "" {
		if _, err := Watch(dir, "Plugboard crash test"); err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			panic("boom in a goroutine")
		}()
		<-done
		return
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestAPanicInAnyGoroutineLandsInTheLog$")
	cmd.Env = append(os.Environ(), "CRASH_TEST_DIR="+dir)
	if err := cmd.Run(); err == nil {
		t.Fatal("the panic didn't end the process")
	}
	data, err := os.ReadFile(filepath.Join(dir, crashFile))
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	if !strings.Contains(log, "--- Plugboard crash test started") || !strings.Contains(log, "panic: boom in a goroutine") || !strings.Contains(log, "goroutine ") {
		t.Errorf("crash.log = %q", log)
	}
	if path, _ := Watch(dir, "Plugboard test"); path == "" {
		t.Error("the next launch didn't notice the crash")
	}
}
