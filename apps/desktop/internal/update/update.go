// Package update finds a newer Relay DB on GitHub releases and installs it.
//
// Every release carries latest.json: the version, its notes and, per
// platform, the URL, SHA-256 and minisign signature of the file to install.
// An update is installed only when the file matches its checksum and carries
// a signature from the key built into the app, whose trusted comment names
// this very version and file — so an old signed release can't be passed off
// as a new one.
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"aead.dev/minisign"
	"github.com/minio/selfupdate"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
	"golang.org/x/mod/semver"
)

// Set at build time with -ldflags -X (see the release workflow).
var (
	// Repo is the GitHub repository releases come from.
	Repo = "relay-client/relay-db"
	// PublicKey is the minisign public key updates must be signed with.
	// A build without one never installs an update.
	PublicKey = ""
)

const (
	metadataTimeout = 15 * time.Second
	downloadTimeout = 10 * time.Minute
	maxManifestSize = 1 << 20
	maxSignature    = 8 << 10
	maxDownload     = 512 << 20
)

var (
	ErrNoPublicKey      = errors.New("this build has no update signing key")
	ErrChecksum         = errors.New("the download doesn't match its checksum")
	ErrSignature        = errors.New("the download isn't signed by Relay DB's release key")
	ErrUntrustedURL     = errors.New("the update points outside Relay DB's GitHub releases")
	ErrNotNewer         = errors.New("the release is not newer than this version")
	ErrNoAsset          = errors.New("the release has no package for this system")
	ErrNotUpdatable     = errors.New("this copy can't update itself: install Relay DB from a release")
	ErrBundleNotRelayDB = errors.New("the downloaded app is not Relay DB")
)

// Manifest is latest.json.
type Manifest struct {
	Version     string              `json:"version"`
	Notes       string              `json:"notes"`
	PublishedAt string              `json:"published_at"`
	Platforms   map[string]Platform `json:"platforms"`
}

// Platform is the file one platform installs.
type Platform struct {
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature"`
}

// Check returns the newest release when it is newer than current, or nil.
func Check(ctx context.Context, current string) (*model.UpdateInfo, error) {
	t, err := currentTarget()
	if err != nil {
		return nil, err
	}
	return check(ctx, current, t)
}

func check(ctx context.Context, current string, target target) (*model.UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	m, err := fetchManifest(ctx)
	if err != nil {
		return nil, err
	}
	if !newer(m.Version, current) {
		return nil, nil
	}
	if _, ok := m.Platforms[target.key]; !ok {
		return nil, nil // released for other systems only; this one waits for the next
	}
	version := strings.TrimPrefix(m.Version, "v")
	return &model.UpdateInfo{
		Version:     version,
		Notes:       m.Notes,
		PublishedAt: m.PublishedAt,
		ReleaseURL:  "https://github.com/" + Repo + "/releases/tag/v" + version,
	}, nil
}

// Install downloads the newest release, verifies it and puts it in place of
// the running app; Restart then switches to it. It returns the installed
// version. The release is looked up afresh rather than taken from the UI.
func Install(ctx context.Context, current string) (string, error) {
	t, err := currentTarget()
	if err != nil {
		return "", err
	}
	return install(ctx, current, t)
}

func install(ctx context.Context, current string, target target) (string, error) {
	if strings.TrimSpace(PublicKey) == "" {
		return "", ErrNoPublicKey
	}
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	m, err := fetchManifest(ctx)
	if err != nil {
		return "", err
	}
	version := strings.TrimPrefix(m.Version, "v")
	if !newer(version, current) {
		return "", ErrNotNewer
	}
	p, ok := m.Platforms[target.key]
	if !ok || strings.TrimSpace(p.URL) == "" {
		return "", ErrNoAsset
	}
	file, err := download(ctx, p.URL, maxDownload)
	if err != nil {
		return "", err
	}
	defer os.Remove(file)
	if err := checkSHA256(file, p.SHA256); err != nil {
		return "", err
	}
	if err := checkSignature(ctx, file, p.Signature, signedComment(version, assetName(p.URL))); err != nil {
		return "", err
	}
	if target.bundle != "" {
		return version, installBundle(file, target.bundle, version)
	}
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return version, selfupdate.Apply(f, selfupdate.Options{TargetPath: target.path})
}

// target is what an update replaces on this system, and the latest.json key
// of the file that replaces it.
type target struct {
	key    string
	path   string // the executable (or AppImage) to replace
	bundle string // or, on macOS, the .app whose contents to replace
}

func currentTarget() (target, error) {
	switch runtime.GOOS {
	case "darwin":
		// The whole app is replaced, so its icon and Info.plist change with it.
		bundle := runningBundle()
		if bundle == "" {
			return target{}, ErrNotUpdatable
		}
		return target{key: "darwin-" + macArch(), bundle: bundle}, nil
	case "linux":
		if appImage := os.Getenv("APPIMAGE"); appImage != "" {
			return target{key: "linux-" + runtime.GOARCH + "-appimage", path: appImage}, nil
		}
	}
	return target{key: runtime.GOOS + "-" + runtime.GOARCH}, nil
}

// newer reports whether version is a later release than current. A dev build
// is older than any release.
func newer(version, current string) bool {
	v := "v" + strings.TrimPrefix(strings.TrimSpace(version), "v")
	c := "v" + strings.TrimPrefix(strings.TrimSpace(current), "v")
	if !semver.IsValid(v) {
		return false
	}
	return !semver.IsValid(c) || semver.Compare(v, c) > 0
}

func manifestURL() string {
	return "https://github.com/" + Repo + "/releases/latest/download/latest.json"
}

// trustedURL is the hook tests use to point downloads at a local server.
var trustedURL = func(u *url.URL) bool {
	if u.Scheme != "https" {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "github.com":
		return strings.HasPrefix(u.Path, "/"+Repo+"/releases/")
	case "objects.githubusercontent.com", "release-assets.githubusercontent.com":
		return true // where GitHub redirects release downloads
	}
	return false
}

var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		if !trustedURL(req.URL) {
			return ErrUntrustedURL
		}
		return nil
	},
}

func get(ctx context.Context, rawURL string) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil || !trustedURL(u) {
		return nil, ErrUntrustedURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s returned %s", assetName(rawURL), resp.Status)
	}
	return resp, nil
}

func fetchManifest(ctx context.Context) (*Manifest, error) {
	resp, err := get(ctx, manifestURL())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var m Manifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxManifestSize)).Decode(&m); err != nil {
		return nil, fmt.Errorf("read latest.json: %w", err)
	}
	if !semver.IsValid("v" + strings.TrimPrefix(m.Version, "v")) {
		return nil, fmt.Errorf("latest.json has no valid version (%q)", m.Version)
	}
	return &m, nil
}

// download saves rawURL to a temporary file and returns its path.
func download(ctx context.Context, rawURL string, limit int64) (string, error) {
	resp, err := get(ctx, rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	tmp, err := os.CreateTemp("", "relay-db-update-*")
	if err != nil {
		return "", err
	}
	n, err := io.Copy(tmp, io.LimitReader(resp.Body, limit+1))
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil && n > limit {
		err = fmt.Errorf("%s is larger than %d MB", assetName(rawURL), limit>>20)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func checkSHA256(file, want string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if want = strings.TrimSpace(want); want == "" || !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), want) {
		return ErrChecksum
	}
	return nil
}

// signedComment is the trusted comment the release workflow signs each file
// with; it ties the signature to one version and one file.
func signedComment(version, asset string) string {
	return "relay-db v" + strings.TrimPrefix(version, "v") + " " + asset
}

func checkSignature(ctx context.Context, file, signatureURL, comment string) error {
	var key minisign.PublicKey
	if err := key.UnmarshalText([]byte(strings.TrimSpace(PublicKey))); err != nil {
		return fmt.Errorf("the built-in update key is invalid: %w", err)
	}
	if strings.TrimSpace(signatureURL) == "" {
		return ErrSignature
	}
	sigFile, err := download(ctx, signatureURL, maxSignature)
	if err != nil {
		return err
	}
	defer os.Remove(sigFile)
	sig, err := os.ReadFile(sigFile)
	if err != nil {
		return err
	}
	var parsed minisign.Signature
	if err := parsed.UnmarshalText(sig); err != nil || parsed.TrustedComment != comment {
		return ErrSignature
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	r := minisign.NewReader(f)
	if _, err := io.Copy(io.Discard, r); err != nil {
		return err
	}
	if !r.Verify(key, sig) {
		return ErrSignature
	}
	return nil
}

func assetName(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		return path.Base(u.Path)
	}
	return rawURL
}
