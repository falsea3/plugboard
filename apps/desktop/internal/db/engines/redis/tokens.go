package redis

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

const hexMark = "\x00hex:"

func token(b string) string {
	if utf8.ValidString(b) && !strings.HasPrefix(b, hexMark) {
		return b
	}
	return hexMark + hex.EncodeToString([]byte(b))
}

func untoken(t string) (string, error) {
	h, ok := strings.CutPrefix(t, hexMark)
	if !ok {
		return t, nil
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return "", fmt.Errorf("bad key token: %w", err)
	}
	return string(b), nil
}

func display(b string) (string, bool) {
	if utf8.ValidString(b) && !strings.ContainsRune(b, 0) {
		return b, false
	}
	var sb strings.Builder
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c >= 0x20 && c < 0x7f && c != '\\' {
			sb.WriteByte(c)
		} else if c == '\\' {
			sb.WriteString(`\\`)
		} else {
			fmt.Fprintf(&sb, `\x%02x`, c)
		}
	}
	return sb.String(), true
}
