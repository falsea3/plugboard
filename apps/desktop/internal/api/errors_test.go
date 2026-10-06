package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestErrorsGetACodeAndAPlainMessage(t *testing.T) {
	_, refused := net.Dial("tcp", "127.0.0.1:1")
	if refused == nil {
		t.Skip("something listens on 127.0.0.1:1")
	}
	cases := []struct {
		err  error
		code string
		says string
	}{
		{context.Canceled, "canceled", "Stopped."},
		{fmt.Errorf("query: %w", context.DeadlineExceeded), "timeout", "The server didn't answer in time."},
		{fmt.Errorf("connect: %w", &net.DNSError{Name: "db.nowhere", Err: "no such host", IsNotFound: true}), "host_unknown", "“db.nowhere”"},
		{fmt.Errorf("failed to connect: %w", refused), "refused", "Nothing answers at 127.0.0.1:1"},
		{fmt.Errorf("tls error: %w", &tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}), "tls_untrusted", "isn't signed by an authority"},
		{tls.RecordHeaderError{Msg: "first record does not look like a TLS handshake"}, "tls_unsupported", "Turn SSL off"},
		{fmt.Errorf("edit: %w", db.ErrConnLost), "conn_lost", "connection to the database was lost"},
		{errConnectionClosed, "closed", "This connection is closed"},
		{fmt.Errorf("open tunnel: %w", &db.ReadOnlyError{Reason: "writes"}), "read_only", "read-only connection: writes are not allowed"},
		{errors.New(`ERROR: relation "nope" does not exist (SQLSTATE 42P01)`), "error", `relation "nope" does not exist`},
	}
	for _, c := range cases {
		got := describe(c.err)
		if got.Code != c.code || !strings.Contains(got.Message, c.says) {
			t.Errorf("%v\n  = %s %q, want %s containing %q", c.err, got.Code, got.Message, c.code, c.says)
		}
		if !strings.EqualFold(got.Message, c.err.Error()) && got.Detail != c.err.Error() {
			t.Errorf("%v: the original text was dropped (detail %q)", c.err, got.Detail)
		}
	}
}

func TestAnyCodedErrorKeepsItsCode(t *testing.T) {
	err := fmt.Errorf("export: %w", apperr.New("disk_full", "there's no space left for the export"))
	got := describe(err)
	if got.Code != "disk_full" || got.Message != "Export: there's no space left for the export" {
		t.Errorf("describe = %+v", got)
	}
}

func TestFormatErrorIsJSONText(t *testing.T) {
	out, ok := FormatError(apperr.New("conn_lost", "lost")).(string)
	if !ok {
		t.Fatalf("FormatError gave %T: the runtime wraps anything but a string as [object Object]", out)
	}
	var got model.AppError
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.Code != "conn_lost" || got.Message != "Lost" {
		t.Fatalf("FormatError = %s (%v)", out, err)
	}
}
