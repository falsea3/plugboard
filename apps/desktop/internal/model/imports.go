package model

type ImportCandidate struct {
	Key        string     `json:"key"`
	Project    string     `json:"project"`
	Connection Connection `json:"connection"`
	Note       string     `json:"note"`
	Skip       string     `json:"skip"`
	Exists     bool       `json:"exists"`
	Password   bool       `json:"password"`
}

type ImportScan struct {
	Root       string            `json:"root"`
	Candidates []ImportCandidate `json:"candidates"`
}

type ImportDone struct {
	Added  int      `json:"added"`
	Failed []string `json:"failed"`
}
