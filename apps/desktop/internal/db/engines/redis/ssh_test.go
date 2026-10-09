package redis_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/redis"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestThroughSSH(t *testing.T) {
	addr := os.Getenv("PLUGBOARD_TEST_SSH")
	if addr == "" || os.Getenv("PLUGBOARD_TEST_REDIS") == "" {
		t.Skip("PLUGBOARD_TEST_SSH and PLUGBOARD_TEST_REDIS not set")
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	c := model.Connection{
		Driver: model.Redis, Host: "redis", Port: 6379, Password: "relay",
		SSH: model.SSHTunnel{Enabled: true, Host: addr[:i], Port: port, User: "relay", Auth: model.SSHAuthPassword, Password: "relay"},
	}
	ctx := context.Background()
	s, err := redis.Open(ctx, "ssh", c, db.OpenOptions{KnownHostsFile: t.TempDir() + "/known_hosts"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	res, err := s.Run(ctx, "GET shop:config:currency")
	if err != nil || res[0].Rows[0][0] != "USD" {
		t.Fatalf("res = %v, err = %v", res, err)
	}
}

func TestWrongPasswordSaysSo(t *testing.T) {
	addr := os.Getenv("PLUGBOARD_TEST_REDIS")
	if addr == "" {
		t.Skip("PLUGBOARD_TEST_REDIS not set")
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	_, err := redis.Open(context.Background(), "x", model.Connection{Driver: model.Redis, Host: addr[:i], Port: port, Password: "nope"}, db.OpenOptions{})
	if err == nil || !strings.Contains(err.Error(), "WRONGPASS") {
		t.Fatalf("wrong password = %v", err)
	}
}
