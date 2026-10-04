package update

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/minio/selfupdate"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"golang.org/x/mod/semver"
)

var (
	Repo      = "relay-client/plugboard"
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
	ErrNoPublicKey        = errors.New("this build has no update signing key")
	ErrChecksum           = errors.New("the download doesn't match its checksum")
	ErrSignature          = errors.New("the download isn't signed by Plugboard's release key")
	ErrUntrustedURL       = errors.New("the update points outside Plugboard's GitHub releases")
	ErrNotNewer           = errors.New("the release is not newer than this version")
	ErrNoAsset            = errors.New("the release has no package for this system")
	ErrNotUpdatable       = errors.New("this copy can't update itself: install Plugboard from a release")
	ErrBundleNotPlugboard = errors.New("the downloaded app is not Plugboard")
)

type Manifest struct {
	Version     string              `json:"version"`
	Notes       string              `json:"notes"`
	PublishedAt string              `json:"published_at"`
	Platforms   map[string]Platform `json:"platforms"`
}

type Platform struct {
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature"`
}

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
		return nil, nil
	}
	version := strings.TrimPrefix(m.Version, "v")
	return &model.UpdateInfo{
		Version:     version,
		Notes:       m.Notes,
		PublishedAt: m.PublishedAt,
		ReleaseURL:  "https://github.com/" + Repo + "/releases/tag/v" + version,
	}, nil
}

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

type target struct {
	key    string
	path   string
	bundle string
}

func currentTarget() (target, error) {
	switch runtime.GOOS {
	case "darwin":
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

func newer(version, current string) bool {
	v := "v" + strings.TrimPrefix(strings.TrimSpace(version), "v")
	c := "v" + strings.TrimPrefix(strings.TrimSpace(current), "v")
	if !semver.IsValid(v) {
		return false
	}
	return !semver.IsValid(c) || semver.Compare(v, c) > 0
}
