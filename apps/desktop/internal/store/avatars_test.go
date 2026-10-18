package store

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"testing"
)

func pngURL(t *testing.T, side int) string {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, side, side))); err != nil {
		t.Fatal(err)
	}
	return avatarPrefix + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestAvatars(t *testing.T) {
	a := NewAvatars(t.TempDir())
	if err := a.Set("abc123", pngURL(t, 128)); err != nil {
		t.Fatal(err)
	}
	all, err := a.All()
	if err != nil || all["abc123"] != pngURL(t, 128) {
		t.Fatalf("all = %v, %v", len(all), err)
	}
	for _, bad := range []struct{ id, url string }{
		{"../escape", pngURL(t, 8)},
		{"abc", "data:image/jpeg;base64,AAAA"},
		{"abc", avatarPrefix + base64.StdEncoding.EncodeToString([]byte("not a png"))},
		{"abc", pngURL(t, 1024)},
	} {
		if a.Set(bad.id, bad.url) == nil {
			t.Errorf("Set(%q) should fail", bad.id)
		}
	}
	if err := a.Delete("abc123"); err != nil {
		t.Fatal(err)
	}
	if all, _ := a.All(); len(all) != 0 {
		t.Fatalf("left after delete: %v", len(all))
	}
	if err := a.Delete("abc123"); err != nil {
		t.Fatalf("deleting twice: %v", err)
	}
}
