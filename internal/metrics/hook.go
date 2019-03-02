package metrics

import (
	"fmt"
	"net"
	"os"
)

// ConnectionLifecycleHook is a metrics hook interface for reporting events that occur during a TCP
// connection lifecycle. Note that it is not pertinent to UDP transports, since it is an inherently
// connectionless protocol.
type ConnectionLifecycleHook interface {
	// EmitConnectionOpen reports the event that a connection was successfully opened.
	EmitConnectionOpen(addr net.Addr)

	// EmitConnectionClose reports the event that a connection was closed.
	EmitConnectionClose(addr net.Addr)

	// EmitConnectionError reports occurrence of an error establishing a connection.
	EmitConnectionError()
}

// AsyncStatsdConnectionLifecycleHook is an implementation of ConnectionLifecycleHook that outputs
// metrics asynchronously to statsd.
type AsyncStatsdConnectionLifecycleHook struct {
	client *StatsdClient
	source string
}

// NewAsyncStatsdConnectionLifecycleHook creates a new client with the specified source, statsd
// address, and statsd sample rate. The source denotes the entity with whom the server is opening
// and closing TCP connections.
func NewAsyncStatsdConnectionLifecycleHook(source string, addr string, sampleRate float32) (ConnectionLifecycleHook, error) {
	client, err := statsdClientFactory(addr, sampleRate)
	if err != nil {
		return nil, err
	}

	return &AsyncStatsdConnectionLifecycleHook{
		client: client,
		source: source,
	}, nil
}

func (h *AsyncStatsdConnectionLifecycleHook) EmitConnectionOpen(addr net.Addr) {
	go h.client.Count(fmt.Sprintf("event.%s.cx_open", h.source), 1, map[string]string{
		"addr":      ipFromAddr(addr),
		"transport": transportFromAddr(addr),
	})
}

func (h *AsyncStatsdConnectionLifecycleHook) EmitConnectionClose(addr net.Addr) {
	go h.client.Count(fmt.Sprintf("event.%s.cx_close", h.source), 1, map[string]string{
		"addr":      ipFromAddr(addr),
		"transport": transportFromAddr(addr),
	})
}

func (h *AsyncStatsdConnectionLifecycleHook) EmitConnectionError() {
	go h.client.Count(fmt.Sprintf("event.%s.cx_error", h.source), 1, nil)
}

// statsdClientFactory creates a configured StatsdClient with reasonable defaults for the given
// statsd server address and sample rate.
func statsdClientFactory(addr string, sampleRate float32) (*StatsdClient, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	defaultTags := map[string]string{
		"host": hostname,
	}

	return NewStatsdClient(addr, "dotproxy", defaultTags, sampleRate)
}

// ipFromAddr returns the IP address from a full net.Addr, or null if unavailable.
func ipFromAddr(addr net.Addr) string {
	switch networkAddr := addr.(type) {
	case *net.UDPAddr:
		return networkAddr.IP.String()
	case *net.TCPAddr:
		return networkAddr.IP.String()
	default:
		return "null"
	}
}

// transportFromAddr returns the transport protocol (as a string) behind a net.Addr, or null if
// unavailable.
func transportFromAddr(addr net.Addr) string {
	switch addr.(type) {
	case *net.UDPAddr:
		return "udp"
	case *net.TCPAddr:
		return "tcp"
	default:
		return "null"
	}
}