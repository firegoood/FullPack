package peerauto

import (
	"context"
	"testing"
	"time"
)

func TestCancelledCarrierJoinsItsFirewallCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entered, release, joined := make(chan struct{}), make(chan struct{}), make(chan struct{})
	stop := afterFuncJoined(ctx, func() { close(entered); <-release })
	cancel()
	<-entered
	go func() { stop(); close(joined) }()
	select {
	case <-joined:
		t.Fatal("carrier owner exited while its cleanup was still running")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	select {
	case <-joined:
	case <-time.After(time.Second):
		t.Fatal("completed cleanup was not joined")
	}
	stop() // completed callback can be joined repeatedly
}

func TestCarrierCanDisarmCleanupBeforeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := afterFuncJoined(ctx, func() { t.Error("disarmed callback ran") })
	stop()
	stop() // repeated socket Close must not wait for a disarmed callback
}
