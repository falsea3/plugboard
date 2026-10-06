package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"unicode"
	"unicode/utf8"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type rule struct {
	code  string
	match func(error) bool
	text  func(error) string
}

var transport = []rule{
	{"canceled", is(context.Canceled), say("Stopped.")},
	{"host_unknown", as[*net.DNSError], func(err error) string {
		return fmt.Sprintf("Can't find the host “%s”. Check the name, or the VPN if the host is internal.", find[*net.DNSError](err).Name)
	}},
	{"refused", isAny(refusedErrnos...), func(err error) string {
		return fmt.Sprintf("Nothing answers at %s. Is the server running, and is the port right?", where(err))
	}},
	{"unreachable", isAny(unreachableErrnos...), func(err error) string {
		return fmt.Sprintf("Can't reach %s from this computer. Check the network or the VPN.", where(err))
	}},
	{"timeout", either(is(context.DeadlineExceeded), is(os.ErrDeadlineExceeded), timedOut), func(err error) string {
		return where(err) + " didn't answer in time."
	}},
	{"tls_untrusted", as[x509.UnknownAuthorityError], say("The server's certificate isn't signed by an authority this computer trusts.")},
	{"tls_hostname", as[x509.HostnameError], say("The server's certificate was issued for another host name.")},
	{"tls_invalid", as[x509.CertificateInvalidError], func(err error) string {
		return "The server's certificate isn't valid: " + find[x509.CertificateInvalidError](err).Error()
	}},
	{"tls_unsupported", as[tls.RecordHeaderError], say("The server didn't answer in TLS. Turn SSL off for this connection, or check the port.")},
}

func FormatError(err error) any {
	data, merr := json.Marshal(describe(err))
	if merr != nil {
		return err.Error()
	}
	return string(data)
}

func describe(err error) model.AppError {
	raw := err.Error()
	if coded, ok := errors.AsType[apperr.Coded](err); ok {
		return result(coded.Code(), raw, raw)
	}
	for _, r := range transport {
		if r.match(err) {
			return result(r.code, r.text(err), raw)
		}
	}
	return result("error", raw, raw)
}

func result(code, text, raw string) model.AppError {
	e := model.AppError{Code: code, Message: sentence(text)}
	if text != raw {
		e.Detail = raw
	}
	return e
}

func sentence(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

func is(target error) func(error) bool {
	return func(err error) bool { return errors.Is(err, target) }
}

func isAny(targets ...error) func(error) bool {
	return func(err error) bool {
		for _, target := range targets {
			if errors.Is(err, target) {
				return true
			}
		}
		return false
	}
}

func as[T error](err error) bool {
	_, ok := errors.AsType[T](err)
	return ok
}

func find[T error](err error) T {
	v, _ := errors.AsType[T](err)
	return v
}

func either(checks ...func(error) bool) func(error) bool {
	return func(err error) bool {
		for _, check := range checks {
			if check(err) {
				return true
			}
		}
		return false
	}
}

func say(text string) func(error) string {
	return func(error) string { return text }
}

func timedOut(err error) bool {
	ne, ok := errors.AsType[net.Error](err)
	return ok && ne.Timeout()
}

func where(err error) string {
	if op, ok := errors.AsType[*net.OpError](err); ok && op.Addr != nil {
		return op.Addr.String()
	}
	return "the server"
}
