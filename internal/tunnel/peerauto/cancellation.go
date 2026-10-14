package peerauto

import "context"

// context.AfterFunc's stop does not wait for an already-running callback.
// Carrier cleanup can remove firewall rules after closing the socket. Join
// that callback before its owning worker reports shutdown as complete.
func afterFuncJoined(ctx context.Context, f func()) func() bool {
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(done); f() })
	return func() bool {
		stopped := stop()
		if stopped {
			close(done) // disarmed callbacks will never close their completion
		} else {
			<-done
		}
		return stopped
	}
}
