package api

import (
	"os"
	"path/filepath"
	"strings"
)

func (a *App) MCPCommand() string {
	exe, err := os.Executable()
	if err != nil {
		exe = "plugboard"
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if strings.ContainsAny(exe, " '\"") {
		exe = `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	}
	return "claude mcp add plugboard -- " + exe + " mcp"
}

func (a *App) MCPLog() string {
	return filepath.Join(DataDir(), "logs", "mcp.log")
}
