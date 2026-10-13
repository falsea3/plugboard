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
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Host       string          `json:"host"`
	Flags      map[string]bool `json:"flags"`
	ConnLimit  int             `json:"connLimit"`
	ValidUntil string          `json:"validUntil"`
	MemberOf   []string        `json:"memberOf"`
	Grants     []string        `json:"grants"`
	Grantable  bool            `json:"grantable"`
	System     bool            `json:"system"`
	Current    bool            `json:"current"`
}

type RoleSpec struct {
	Role
	Password      string `json:"password"`
	ClearPassword bool   `json:"clearPassword"`
}

type RoleFlag struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Hint  string `json:"hint"`
	Badge string `json:"badge"`
}

type RoleForm struct {
	Noun       string     `json:"noun"`
	Flags      []RoleFlag `json:"flags"`
	Host       bool       `json:"host"`
	ValidUntil bool       `json:"validUntil"`
	New        Role       `json:"new"`
}

type RoleList struct {
	Form  RoleForm `json:"form"`
	Roles []Role   `json:"roles"`
}
