package update

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"aead.dev/minisign"
)

type release struct {
	t       *testing.T
	key     minisign.PrivateKey
	files   map[string][]byte
	version string
	notes   string
}

func newRelease(t *testing.T, version string) *release {
	t.Helper()
	pub, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := pub.MarshalText()
	old := PublicKey
	PublicKey = string(text)
	t.Cleanup(func() { PublicKey = old })
	return &release{t: t, key: priv, files: map[string][]byte{}, version: version, notes: "- Faster"}
}

func (r *release) add(name string, data []byte, comment string) {
	r.files[name] = data
	reader := minisign.NewReader(strings.NewReader(string(data)))
	if _, err := io.Copy(io.Discard, reader); err != nil {
		r.t.Fatal(err)
	}
	r.files[name+".minisig"] = reader.SignWithComments(r.key, comment, "test")
}

func (r *release) manifest(key, name string) []byte {
	sum := sha256.Sum256(r.files[name])
	base := "https://github.com/" + Repo + "/releases/download/v" + r.version + "/"
	m := Manifest{Version: r.version, Notes: r.notes, Platforms: map[string]Platform{
		key: {URL: base + name, SHA256: hex.EncodeToString(sum[:]), Signature: base + name + ".minisig"},
	}}
	data, _ := json.Marshal(m)
	return data
}

func (r *release) serve(manifest []byte) {
	t := r.t
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		name := filepath.Base(req.URL.Path)
		if name == "latest.json" {
			w.Write(manifest)
			return
		}
		data, ok := r.files[name]
		if !ok {
			http.NotFound(w, req)
			return
		}
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	target, _ := url.Parse(srv.URL)
	inner := srv.Client().Transport
	oldTransport := client.Transport
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Host = target.Host
		return inner.RoundTrip(req)
	})
	t.Cleanup(func() { client.Transport = oldTransport })
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewer(t *testing.T) {
	for _, tc := range []struct {
		version, current string
		want             bool
	}{
		{"0.2.0", "0.1.9", true},
		{"v0.10.0", "0.9.0", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.2.0", false},
		{"0.1.0", "dev", true},
		{"0.1.0", "0.1.0-3-gabc-dirty", true},
		{"garbage", "0.1.0", false},
	} {
		if got := newer(tc.version, tc.current); got != tc.want {
			t.Errorf("newer(%q, %q) = %v", tc.version, tc.current, got)
		}
	}
}

func TestInstallReplacesTheFile(t *testing.T) {
	r := newRelease(t, "0.2.0")
	r.add("relay-db-test", []byte("new build"), signedComment("0.2.0", "relay-db-test"))
	r.serve(r.manifest("test", "relay-db-test"))

	exe := filepath.Join(t.TempDir(), "relay-db")
	os.WriteFile(exe, []byte("old build"), 0o755)
	v, err := install(context.Background(), "0.1.0", target{key: "test", path: exe})
	if err != nil || v != "0.2.0" {
		t.Fatalf("install: %q %v", v, err)
	}
	if data, _ := os.ReadFile(exe); string(data) != "new build" {
		t.Fatalf("file holds %q", data)
	}
}

func TestInstallRefuses(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "relay-db")
	for _, tc := range []struct {
		name    string
		setup   func(r *release) []byte
		current string
		want    error
	}{
		{"not newer", func(r *release) []byte {
			r.add("f", []byte("x"), signedComment("0.2.0", "f"))
			return r.manifest("test", "f")
		}, "0.2.0", ErrNotNewer},
		{"bad checksum", func(r *release) []byte {
			r.add("f", []byte("x"), signedComment("0.2.0", "f"))
			m := r.manifest("test", "f")
			r.files["f"] = []byte("tampered")
			return m
		}, "0.1.0", ErrChecksum},
		{"signature for another version", func(r *release) []byte {
			r.add("f", []byte("x"), signedComment("0.1.5", "f"))
			return r.manifest("test", "f")
		}, "0.1.0", ErrSignature},
		{"no package for this system", func(r *release) []byte {
			r.add("f", []byte("x"), signedComment("0.2.0", "f"))
			return r.manifest("other", "f")
		}, "0.1.0", ErrNoAsset},
		{"download elsewhere", func(r *release) []byte {
			return []byte(`{"version":"0.2.0","platforms":{"test":{"url":"https://example.com/f","sha256":"00","signature":"https://example.com/f.minisig"}}}`)
		}, "0.1.0", ErrUntrustedURL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os.WriteFile(exe, []byte("old build"), 0o755)
			r := newRelease(t, "0.2.0")
			r.serve(tc.setup(r))
			if _, err := install(context.Background(), tc.current, target{key: "test", path: exe}); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if data, _ := os.ReadFile(exe); string(data) != "old build" {
				t.Fatalf("file was replaced: %q", data)
			}
		})
	}
}

func TestInstallNeedsAKey(t *testing.T) {
	old := PublicKey
	PublicKey = ""
	defer func() { PublicKey = old }()
	if _, err := install(context.Background(), "0.1.0", target{key: "test"}); !errors.Is(err, ErrNoPublicKey) {
		t.Fatalf("err = %v", err)
	}
}

func TestCheck(t *testing.T) {
	r := newRelease(t, "0.2.0")
	r.add("f", []byte("x"), signedComment("0.2.0", "f"))
	r.serve(r.manifest("test", "f"))
	here := target{key: "test"}
	info, err := check(context.Background(), "0.1.0", here)
	if err != nil || info == nil || info.Version != "0.2.0" || info.Notes != "- Faster" {
		t.Fatalf("check: %+v %v", info, err)
	}
	if info, err := check(context.Background(), "0.2.0", here); err != nil || info != nil {
		t.Fatalf("up to date: %+v %v", info, err)
	}
	if info, err := check(context.Background(), "0.1.0", target{key: "other"}); err != nil || info != nil {
		t.Fatalf("released for other systems only: %+v %v", info, err)
	}
}

func TestInstallBundle(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("app bundles are macOS-only")
	}
	dir := t.TempDir()
	makeApp := func(path, version, exe string) {
		os.MkdirAll(filepath.Join(path, "Contents", "MacOS"), 0o755)
		os.WriteFile(filepath.Join(path, "Contents", "MacOS", "relay-db"), []byte(exe), 0o755)
		plist := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>CFBundleShortVersionString</key><string>` + version + `</string></dict></plist>`
		os.WriteFile(filepath.Join(path, "Contents", "Info.plist"), []byte(plist), 0o644)
	}
	installed := filepath.Join(dir, "Relay DB.app")
	makeApp(installed, "0.1.0", "old")
	makeApp(filepath.Join(dir, "new", "Relay DB.app"), "0.2.0", "new")
	archive := filepath.Join(dir, "update.app.zip")
	if out, err := exec.Command("ditto", "-c", "-k", "--keepParent", filepath.Join(dir, "new", "Relay DB.app"), archive).CombinedOutput(); err != nil {
		t.Fatalf("zip: %v %s", err, out)
	}

	if err := installBundle(archive, installed, "0.3.0"); !errors.Is(err, ErrBundleNotRelayDB) {
		t.Fatalf("wrong version: err = %v", err)
	}
	if err := installBundle(archive, installed, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(installed, "Contents", "MacOS", "relay-db")); string(data) != "new" {
		t.Fatalf("executable = %q", data)
	}
	if entries, _ := os.ReadDir(installed); len(entries) != 1 {
		t.Fatalf("staging left behind: %v", entries)
	}
}

func TestBundleOf(t *testing.T) {
	if got := bundleOf("/Applications/Relay DB.app/Contents/MacOS/relay-db"); got != "/Applications/Relay DB.app" {
		t.Fatalf("got %q", got)
	}
	if got := bundleOf("/usr/local/bin/relay-db"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestReleaseCoversEveryMac(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	manifest, err := os.ReadFile(filepath.Join(root, "scripts", "make-latest-json.py"))
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, arch := range []string{"arm64", "amd64"} {
		file := "relay-db-darwin-" + arch + ".app.zip"
		if !strings.Contains(string(manifest), `"darwin-`+arch+`": "`+file+`"`) {
			t.Errorf("make-latest-json.py has no darwin-%s → %s", arch, file)
		}
		if !strings.Contains(string(workflow), file) {
			t.Errorf("release.yml doesn't sign %s", file)
		}
	}
	if a := macArch(); a != runtime.GOARCH && !(runtime.GOOS == "darwin" && a == "arm64") {
		t.Errorf("macArch() = %s on %s/%s", a, runtime.GOOS, runtime.GOARCH)
	}
}
