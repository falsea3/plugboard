package api

import (
	"context"
	"errors"
	"net"
	"os"
	goruntime "runtime"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"github.com/relay-client/plugboard/apps/desktop/internal/update"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) AppInfo() model.AppInfo {
	return model.AppInfo{
		Name:      "Plugboard",
		Version:   appVersion,
		GoVersion: goruntime.Version(),
		Platform:  goruntime.GOOS + "/" + goruntime.GOARCH,
		DataDir:   DataDir(),
		Copyright: "© 2026 Relay Client",
	}
}

func devBuild() bool {
	v := strings.TrimSpace(appVersion)
	return v == "" || v == "dev" || strings.Contains(v, "-")
}

func (a *App) CheckForUpdate() model.UpdateCheck {
	if devBuild() {
		return model.UpdateCheck{Error: "This is a development build (" + appVersion + "); it doesn't update itself."}
	}
	info, err := update.Check(a.context(), appVersion)
	if err != nil {
		return model.UpdateCheck{Error: updateError("check for updates", err)}
	}
	return model.UpdateCheck{Available: info}
}

func (a *App) InstallUpdate() (string, error) {
	if devBuild() {
		return "", errors.New("development builds don't update themselves")
	}
	version, err := update.Install(a.context(), appVersion)
	if err != nil {
		return "", errors.New(updateError("install the update", err))
	}
	return version, nil
}

func (a *App) RestartApp() error {
	return update.Restart(func() { runtime.Quit(a.context()) })
}

func updateError(action string, err error) string {
	var dnsErr *net.DNSError
	switch {
	case errors.As(err, &dnsErr), errors.Is(err, context.DeadlineExceeded):
		return "Could not " + action + ": GitHub can't be reached. Check the internet connection."
	case errors.Is(err, update.ErrChecksum), errors.Is(err, update.ErrSignature), errors.Is(err, update.ErrUntrustedURL), errors.Is(err, update.ErrBundleNotPlugboard):
		return "Could not " + action + ": the download failed verification (" + err.Error() + "), so nothing was changed."
	case errors.Is(err, os.ErrPermission):
		return "Could not " + action + ": Plugboard may not replace itself where it is installed. Download the new version from GitHub instead."
	}
	return "Could not " + action + ": " + describe(err).Message
}
