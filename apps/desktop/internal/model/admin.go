package model

type ConfigFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type ConfigText struct {
	Path string `json:"path"`
	Text string `json:"text"`
	Via  string `json:"via"`
}

type ConfigSaved struct {
	Via      string          `json:"via"`
	Reloaded bool            `json:"reloaded"`
	Problems []ConfigProblem `json:"problems"`
	Restart  []string        `json:"restart"`
}

type ConfigProblem struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type Role struct {
	Name        string   `json:"name"`
	Login       bool     `json:"login"`
	Superuser   bool     `json:"superuser"`
	CreateDB    bool     `json:"createDB"`
	CreateRole  bool     `json:"createRole"`
	Inherit     bool     `json:"inherit"`
	Replication bool     `json:"replication"`
	BypassRLS   bool     `json:"bypassRLS"`
	ConnLimit   int      `json:"connLimit"`
	ValidUntil  string   `json:"validUntil"`
	MemberOf    []string `json:"memberOf"`
	System      bool     `json:"system"`
	Current     bool     `json:"current"`
}

type RoleSpec struct {
	Role
	Password      string `json:"password"`
	ClearPassword bool   `json:"clearPassword"`
}
