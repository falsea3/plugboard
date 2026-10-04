package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"aead.dev/minisign"
)

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

func signedComment(version, asset string) string {
	return "plugboard v" + strings.TrimPrefix(version, "v") + " " + asset
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
