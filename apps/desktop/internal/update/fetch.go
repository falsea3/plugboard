package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"golang.org/x/mod/semver"
)

func manifestURL() string {
	return "https://github.com/" + Repo + "/releases/latest/download/latest.json"
}

var trustedURL = func(u *url.URL) bool {
	if u.Scheme != "https" {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "github.com":
		return strings.HasPrefix(u.Path, "/"+Repo+"/releases/")
	case "objects.githubusercontent.com", "release-assets.githubusercontent.com":
		return true
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

func download(ctx context.Context, rawURL string, limit int64) (string, error) {
	resp, err := get(ctx, rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	tmp, err := os.CreateTemp("", "plugboard-update-*")
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

func assetName(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		return path.Base(u.Path)
	}
	return rawURL
}
