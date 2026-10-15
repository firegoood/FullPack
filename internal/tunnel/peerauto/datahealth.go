package peerauto

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

var (
	errNoDataRoute  = errors.New("NO_ROUTE: no committed peer route")
	errDataPool     = errors.New("peer stream pool exhausted")
	errBackend      = errors.New("peer backend unavailable (carrier remains healthy)")
	errPeerCapacity = errors.New("peer admission capacity exhausted")
)

type DataStatus struct {
	Code           string            `json:"code"`
	Detail         string            `json:"detail"`
	LastErrorCode  string            `json:"lastErrorCode,omitempty"`
	LastErrorStage string            `json:"lastErrorStage,omitempty"`
	LastErrorAt    time.Time         `json:"lastErrorAt,omitempty"`
	Errors         map[string]uint64 `json:"errors"`
}

type dataDiagnostics struct {
	mu          sync.Mutex
	errors      map[string]uint64 // fixed classifier vocabulary, never user input
	code, stage string
	at, logged  time.Time
}

func dataErrorCode(err error) string {
	var timeout net.Error
	switch {
	case localResourceError(err):
		return "local_resource"
	case errors.Is(err, errAdmissionFull):
		return "queue_full"
	case errors.Is(err, errAdmissionTimeout):
		return "queue_timeout"
	case errors.Is(err, errUDPCapacity):
		return "udp_capacity"
	case errors.Is(err, errUDPHandshakeBusy):
		return "handshake_busy"
	case errors.Is(err, errPeerCapacity):
		return "peer_capacity"
	case errors.Is(err, errBackend):
		return "backend_unavailable"
	case errors.Is(err, errNoDataRoute):
		return "no_route"
	case errors.Is(err, errDataPool):
		return "stream_pool_timeout"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &timeout) && timeout.Timeout():
		return "handshake_timeout"
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, net.ErrClosed):
		return "peer_closed"
	default:
		return "stream_unavailable"
	}
}

func (m *Manager) dataFailure(stage string, err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	now, code := time.Now(), dataErrorCode(err)
	d := &m.dataDiagnostics
	d.mu.Lock()
	if d.errors == nil {
		d.errors = make(map[string]uint64)
	}
	d.errors[code]++
	d.code, d.stage, d.at = code, stage, now
	log := now.Sub(d.logged) >= 10*time.Second
	if log {
		d.logged = now
	}
	d.mu.Unlock()
	if log {
		m.log.WithField("errorCode", code).WithField("stage", stage).Warn("Auto data admission failed; inspect auto status; carrier policy unchanged")
	}
}

func (m *Manager) dataSnapshot(f FlowStatus, now time.Time) *DataStatus {
	d := &m.dataDiagnostics
	d.mu.Lock()
	defer d.mu.Unlock()
	s := &DataStatus{Code: "available", Detail: "data admission available; application delivery is not verified", LastErrorCode: d.code, LastErrorStage: d.stage, LastErrorAt: d.at, Errors: make(map[string]uint64, len(d.errors))}
	for k, v := range d.errors {
		s.Errors[k] = v
	}
	switch {
	case f.Limit > 0 && f.Live >= f.Limit:
		s.Code, s.Detail = "capacity", fmt.Sprintf("connection capacity full (%d/%d); %d TCP connection(s) waiting; existing sessions retained", f.Live, f.Limit, f.QueuedTCP)
	case f.QueuedTCP > 0:
		s.Code, s.Detail = "queueing", fmt.Sprintf("%d TCP connection(s) waiting for capacity (maximum %d, deadline 2s)", f.QueuedTCP, f.QueueLimit)
	case !d.at.IsZero() && now.Sub(d.at) < 30*time.Second:
		s.Code, s.Detail = "recent_errors", "recent data admission failures: "+d.code+"; inspect counters and capacity"
	}
	return s
}

func localResourceError(err error) bool {
	return errors.Is(err, syscall.EMFILE) || errors.Is(err, syscall.ENFILE) || errors.Is(err, syscall.ENOBUFS) || errors.Is(err, syscall.ENOMEM)
}
