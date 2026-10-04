package model

type AppInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
	DataDir   string `json:"dataDir"`
	Copyright string `json:"copyright"`
}

type UpdateInfo struct {
	Version     string `json:"version"`
	Notes       string `json:"notes"`
	PublishedAt string `json:"publishedAt"`
	ReleaseURL  string `json:"releaseUrl"`
}

type UpdateCheck struct {
	Available *UpdateInfo `json:"available,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type Theme string

const (
	ThemeSystem Theme = "system"
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
)

type Settings struct {
	Theme             Theme `json:"theme"`
	PageSize          int   `json:"pageSize"`
	EditorFontSize    int   `json:"editorFontSize"`
	ConfirmProdWrites bool  `json:"confirmProdWrites"`
	AutoUpdate        bool  `json:"autoUpdate"`
}

func DefaultSettings() Settings {
	return Settings{Theme: ThemeSystem, PageSize: 300, EditorFontSize: 13, ConfirmProdWrites: true}
}
