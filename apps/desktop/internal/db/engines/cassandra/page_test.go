package cassandra

import "testing"

func TestCQLCondition(t *testing.T) {
	for _, ok := range []string{"amount > 10 AND status = 'paid'", "note = 'a;b -- c'", "name = 'it''s'"} {
		if got, err := cqlCondition("  " + ok + " "); err != nil || got != ok {
			t.Errorf("%q: %q %v", ok, got, err)
		}
	}
	for _, bad := range []string{"", "a = 1; drop table t", "a = 1 -- x", "a = 1 /* x */", "a = 'open"} {
		if _, err := cqlCondition(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}
