package store

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	avatarPrefix  = "data:image/png;base64,"
	avatarMaxSize = 1 << 20
	avatarMaxSide = 512
)

var avatarID = regexp.MustCompile(`^[0-9a-zA-Z_-]{1,64}$`)

type Avatars struct {
	dir string
}

func NewAvatars(dir string) *Avatars {
	return &Avatars{dir: filepath.Join(dir, "avatars")}
}

func (a *Avatars) path(id string) (string, error) {
	if !avatarID.MatchString(id) {
		return "", errors.New("bad connection id")
	}
	return filepath.Join(a.dir, id+".png"), nil
}

func (a *Avatars) All() (map[string]string, error) {
	out := map[string]string{}
	files, err := filepath.Glob(filepath.Join(a.dir, "*.png"))
	if err != nil {
		return out, err
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return out, err
		}
		out[strings.TrimSuffix(filepath.Base(f), ".png")] = avatarPrefix + base64.StdEncoding.EncodeToString(b)
	}
	return out, nil
}

func (a *Avatars) Set(id, dataURL string) error {
	path, err := a.path(id)
	if err != nil {
		return err
	}
	raw, ok := strings.CutPrefix(dataURL, avatarPrefix)
	if !ok {
		return errors.New("the picture must arrive as a PNG")
	}
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	if len(b) > avatarMaxSize {
		return errors.New("the picture is larger than 1 MB")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return errors.New("the picture isn't a PNG: " + err.Error())
	}
	if cfg.Width > avatarMaxSide || cfg.Height > avatarMaxSide {
		return errors.New("the picture is larger than 512×512")
	}
	return writeFileAtomic(path, b, 0o600)
}

func (a *Avatars) Delete(id string) error {
	path, err := a.path(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
