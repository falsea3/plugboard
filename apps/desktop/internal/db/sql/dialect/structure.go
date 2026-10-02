package dialect

import (
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TypeChanged(cur model.Column, ch model.ColumnChange) bool {
	return ch.Type != nil && !strings.EqualFold(strings.TrimSpace(*ch.Type), cur.Type)
}

func NameOf(cur model.Column, ch model.ColumnChange) string {
	if ch.Name != nil {
		return strings.TrimSpace(*ch.Name)
	}
	return cur.Name
}
