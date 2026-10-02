package update

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const lsregister = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

func runningBundle() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return bundleOf(exe)
}

func bundleOf(exe string) string {
	i := strings.Index(exe, ".app/Contents/MacOS/")
	if i < 0 {
		return ""
	}
	return exe[:i+len(".app")]
}

func installBundle(archive, bundle, version string) error {
	stage, err := os.MkdirTemp(bundle, ".plugboard-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if out, err := exec.Command("ditto", "-x", "-k", archive, stage).CombinedOutput(); err != nil {
		return fmt.Errorf("unpack the update: %v: %s", err, strings.TrimSpace(string(out)))
	}
	contents, err := stagedContents(stage)
	if err != nil {
		return err
	}
	got, err := bundleVersion(filepath.Join(contents, "Info.plist"))
	if err != nil {
		return err
	}
	if got != version {
		return fmt.Errorf("%w: it says version %s, the release %s", ErrBundleNotPlugboard, got, version)
	}
	current := filepath.Join(bundle, "Contents")
	retired := filepath.Join(stage, "old-Contents")
	if err := os.Rename(current, retired); err != nil {
		return err
	}
	if err := os.Rename(contents, current); err != nil {
		if restoreErr := os.Rename(retired, current); restoreErr != nil {
			return errors.Join(err, fmt.Errorf("the previous version could not be put back: %w", restoreErr))
		}
		return err
	}
	now := time.Now()
	_ = os.Chtimes(bundle, now, now)
	_ = exec.Command(lsregister, "-f", bundle).Run()
	return nil
}

func stagedContents(stage string) (string, error) {
	entries, err := os.ReadDir(stage)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasSuffix(e.Name(), ".app") {
			continue
		}
		contents := filepath.Join(stage, e.Name(), "Contents")
		if info, err := os.Stat(filepath.Join(contents, "Info.plist")); err != nil || !info.Mode().IsRegular() {
			continue
		}
		if exes, err := os.ReadDir(filepath.Join(contents, "MacOS")); err != nil || len(exes) == 0 {
			continue
		}
		return contents, nil
	}
	return "", ErrBundleNotPlugboard
}

func bundleVersion(plist string) (string, error) {
	out, err := exec.Command("plutil", "-extract", "CFBundleShortVersionString", "raw", "-o", "-", plist).Output()
	if err != nil {
		return "", fmt.Errorf("read the app's version: %w", err)
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "v"), nil
}
