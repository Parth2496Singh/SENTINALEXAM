package jetstream

import (
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/sentinelexam/backend/internal/platform/config"
)

// NATSClient encapsulates the NATS connection and JetStream context.
type NATSClient struct {
	Conn   *nats.Conn
	JS     nats.JetStreamContext
	logger *slog.Logger
}

// NewClient initializes a connection to NATS and binds the JetStream context.
func NewClient(cfg *config.Config, logger *slog.Logger) (*NATSClient, error) {
	opts := []nats.Option{
		nats.MaxReconnects(cfg.NATSMaxReconnects),
		nats.ReconnectWait(cfg.NATSReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("nats reconnected", "url", nc.ConnectedUrl())
		}),
	}

	nc, err := nats.Connect(cfg.NATSURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats at %s: %w", cfg.NATSURL, err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize jetstream context: %w", err)
	}

	return &NATSClient{
		Conn:   nc,
		JS:     js,
		logger: logger,
	}, nil
}

// IsConnected checks if the NATS connection is active.
func (n *NATSClient) IsConnected() bool {
	return n.Conn != nil && n.Conn.IsConnected()
}

// Close gracefully closes the NATS connection.
func (n *NATSClient) Close() {
	if n.Conn != nil {
		n.logger.Info("closing nats connection")
		n.Conn.Close()
	}
}
