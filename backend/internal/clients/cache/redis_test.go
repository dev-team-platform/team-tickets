package cache

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisTestDialer provides just enough of the Redis protocol for the client's
// handshake and ping, without opening a listening socket.
func redisTestDialer(pingReply string, commands chan<- string) func(context.Context, string, string) (net.Conn, error) {
	return func(_ context.Context, _, _ string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			reader := bufio.NewReader(server)
			for {
				line, err := reader.ReadString('\n')
				if err != nil || !strings.HasPrefix(line, "*") {
					return
				}
				count, err := strconv.Atoi(strings.TrimSpace(line[1:]))
				if err != nil || count < 1 {
					return
				}
				args := make([]string, 0, count)
				for i := 0; i < count; i++ {
					line, err = reader.ReadString('\n')
					if err != nil || !strings.HasPrefix(line, "$") {
						return
					}
					size, err := strconv.Atoi(strings.TrimSpace(line[1:]))
					if err != nil || size < 0 {
						return
					}
					data := make([]byte, size+2)
					if _, err := io.ReadFull(reader, data); err != nil {
						return
					}
					args = append(args, string(data[:size]))
				}
				command := strings.ToUpper(args[0])
				if commands != nil {
					select {
					case commands <- command:
					default:
					}
				}
				reply := "+OK\r\n"
				switch command {
				case "HELLO":
					reply = "-ERR unknown command 'HELLO'\r\n"
				case "PING":
					reply = pingReply
				}
				if _, err := fmt.Fprint(server, reply); err != nil {
					return
				}
			}
		}()
		return client, nil
	}
}

func TestConnectRedisReturnsClientAfterSuccessfulPing(t *testing.T) {
	commands := make(chan string, 10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, err := connectRedis(ctx, &redis.Options{Addr: "redis:6379", Password: "password", Dialer: redisTestDialer("+PONG\r\n", commands)})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	seen := map[string]bool{}
	for len(commands) > 0 {
		seen[<-commands] = true
	}
	if !seen["AUTH"] || !seen["PING"] {
		t.Fatalf("expected authentication and ping, got %v", seen)
	}
}

func TestConnectRedisReturnsErrorWhenPingFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, err := connectRedis(ctx, &redis.Options{Addr: "redis:6379", Dialer: redisTestDialer("-ERR ping failed\r\n", nil)})
	if client != nil || err == nil || !strings.Contains(err.Error(), "ping redis") {
		t.Fatalf("expected wrapped ping error and no client, got %v, %v", client, err)
	}
}
