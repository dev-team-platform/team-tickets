package messaging

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/trannhanlv2004/team-tickets/internal/config"
)

// Connection reuses a RabbitMQ connection and reconnects on the next request
// after a disconnect. Channels must be closed by their callers.
type Connection struct {
	mu     sync.Mutex
	cfg    config.RabbitMQConfig
	conn   *amqp.Connection
	closed bool
}

func NewConnection(cfg config.RabbitMQConfig) (*Connection, error) {
	if strings.TrimSpace(cfg.Host) == "" || cfg.Port < 1 || cfg.Port > 65535 || strings.TrimSpace(cfg.User) == "" || cfg.Password == "" || cfg.VHost == "" {
		return nil, errors.New("RabbitMQ host, port, user, password and virtual host are required")
	}
	return &Connection{
		cfg: cfg,
	}, nil
}

func (c *Connection) Get(ctx context.Context) (*amqp.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errors.New("RabbitMQ connection is closed")
	}
	if c.conn != nil && !c.conn.IsClosed() {
		return c.conn, nil
	}
	addr := net.JoinHostPort(c.cfg.Host, strconv.Itoa(c.cfg.Port))
	var transport net.Conn
	var stopCancel func() bool
	conn, err := amqp.DialConfig("amqp://"+addr+"/", amqp.Config{
		SASL: []amqp.Authentication{
			&amqp.PlainAuth{
				Username: c.cfg.User,
				Password: c.cfg.Password,
			},
		},
		Vhost:     c.cfg.VHost,
		Heartbeat: 10 * time.Second,
		Dial: func(network, address string) (net.Conn, error) {
			conn, err := (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			transport = conn
			stopCancel = context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Now()) })
			deadline := time.Now().Add(10 * time.Second)
			if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
				deadline = ctxDeadline
			}
			if err := conn.SetDeadline(deadline); err != nil {
				_ = conn.Close()
				return nil, err
			}
			return conn, nil
		},
	})
	if stopCancel != nil {
		stopCancel()
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("connect RabbitMQ: %w", err)
	}
	if err := transport.SetDeadline(time.Time{}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	c.conn = conn
	return conn, nil
}

func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.conn != nil && !c.conn.IsClosed() {
		return c.conn.Close()
	}
	return nil
}
