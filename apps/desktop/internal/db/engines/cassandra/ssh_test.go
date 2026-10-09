package cassandra_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/cassandra"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestThroughSSH(t *testing.T) {
	addr := os.Getenv("PLUGBOARD_TEST_SSH")
	if addr == "" || os.Getenv("PLUGBOARD_TEST_CASSANDRA") == "" {
		t.Skip("PLUGBOARD_TEST_SSH and PLUGBOARD_TEST_CASSANDRA not set")
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	c := model.Connection{
		Driver: model.Cassandra, Host: "cassandra", Port: 9042, User: "relay", Password: "relay", Database: "shop",
		SSH: model.SSHTunnel{Enabled: true, Host: addr[:i], Port: port, User: "relay", Auth: model.SSHAuthPassword, Password: "relay"},
	}
	s, err := cassandra.Open(context.Background(), "ssh", c, db.OpenOptions{KnownHostsFile: t.TempDir() + "/known_hosts"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	res, err := s.Run(context.Background(), "SELECT name FROM customers WHERE id = 1")
	if err != nil || res[0].Rows[0][0] != "Ann Novak" {
		t.Fatalf("res = %v, err = %v", res, err)
	}
}
