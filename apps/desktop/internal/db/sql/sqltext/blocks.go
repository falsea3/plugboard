package sqltext

import "strings"

var routineKinds = map[string]bool{"TRIGGER": true, "PROCEDURE": true, "FUNCTION": true, "EVENT": true}

var endsOther = map[string]bool{"IF": true, "LOOP": true, "WHILE": true, "REPEAT": true}

type blocks struct {
	words   int
	create  bool
	begin   bool
	routine bool
	batch   bool
	depth   int
	prev    string
}

func (b *blocks) word(w string) {
	u := strings.ToUpper(w)
	b.words++
	defer func() { b.prev = u }()
	if b.words == 1 {
		b.create = u == "CREATE"
		b.begin = u == "BEGIN"
		return
	}
	if b.begin && b.words <= 3 && u == "BATCH" {
		b.batch = true
		return
	}
	if b.batch {
		if b.prev == "APPLY" && u == "BATCH" {
			b.batch, b.begin = false, false
		}
		return
	}
	if !b.routine {
		b.routine = b.create && b.words <= 8 && routineKinds[u]
		return
	}
	switch {
	case u == "END":
		b.depth--
	case b.prev == "END" && endsOther[u]:
		b.depth++
	case u == "BEGIN", u == "CASE" && b.prev != "END":
		b.depth++
	}
}

func (b *blocks) open() bool { return b.routine && b.depth > 0 || b.batch }

func delimiterAt(s string, i int) (delim string, next int, ok bool) {
	const kw = "DELIMITER"
	if len(s)-i < len(kw)+2 || !strings.EqualFold(s[i:i+len(kw)], kw) || (s[i+len(kw)] != ' ' && s[i+len(kw)] != '\t') {
		return "", i, false
	}
	end := strings.IndexByte(s[i:], '\n')
	if end < 0 {
		end = len(s)
	} else {
		end += i
	}
	delim = strings.TrimSpace(s[i+len(kw) : end])
	if delim == "" || strings.ContainsAny(delim, " \t") {
		return "", i, false
	}
	return delim, end, true
}
