package api

import (
	"fmt"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/importer/dbeaver"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
	"github.com/falsea3/plugboard/apps/desktop/internal/store"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) FindDBeaver() (model.ImportScan, error) {
	roots := dbeaver.Workspaces()
	if len(roots) == 0 {
		return model.ImportScan{Candidates: []model.ImportCandidate{}}, nil
	}
	return a.scanDBeaver(roots[0])
}

func (a *App) ChooseDBeaverFolder() (model.ImportScan, error) {
	dir, err := runtime.OpenDirectoryDialog(a.context(), runtime.OpenDialogOptions{Title: "DBeaver workspace or project folder", ShowHiddenFiles: true})
	if err != nil || dir == "" {
		return model.ImportScan{Candidates: []model.ImportCandidate{}}, err
	}
	return a.scanDBeaver(dir)
}

func (a *App) scanDBeaver(root string) (model.ImportScan, error) {
	found, err := dbeaver.Read(root)
	if err != nil {
		return model.ImportScan{}, err
	}
	have, err := a.connections.List()
	if err != nil {
		return model.ImportScan{}, err
	}
	scan := model.ImportScan{Root: root, Candidates: make([]model.ImportCandidate, 0, len(found))}
	for _, f := range found {
		c := model.ImportCandidate{Key: f.Key, Project: f.Project, Note: f.Note, Skip: f.Skip, Connection: f.Connection.WithoutSecrets()}
		c.Password = f.Connection.Password != "" || f.Connection.SSH.Password != "" || f.Connection.SSH.Passphrase != ""
		if c.Skip == "" {
			if err := store.Validate(f.Connection); err != nil {
				c.Skip = err.Error()
			}
		}
		for _, h := range have {
			c.Exists = c.Exists || sameTarget(h, f.Connection)
		}
		scan.Candidates = append(scan.Candidates, c)
	}
	return scan, nil
}

func sameTarget(a, b model.Connection) bool {
	return a.Driver == b.Driver && sameHost(a.Host, b.Host) && a.Port == b.Port && a.Database == b.Database && a.User == b.User && a.File == b.File
}

func sameHost(a, b string) bool {
	local := func(h string) bool { return h == "localhost" || h == "127.0.0.1" || h == "::1" }
	a, b = strings.ToLower(a), strings.ToLower(b)
	return a == b || local(a) && local(b)
}

func (a *App) ImportDBeaver(root string, keys []string) (model.ImportDone, error) {
	found, err := dbeaver.Read(root)
	if err != nil {
		return model.ImportDone{}, err
	}
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	done := model.ImportDone{Failed: []string{}}
	for _, f := range found {
		if !want[f.Key] || f.Skip != "" {
			continue
		}
		if _, err := a.connections.Save(f.Connection); err != nil {
			done.Failed = append(done.Failed, fmt.Sprintf("%s: %s", f.Connection.Name, err))
			continue
		}
		done.Added++
	}
	return done, nil
}
