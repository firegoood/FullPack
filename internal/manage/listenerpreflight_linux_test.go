//go:build linux

package manage

import (
	"errors"
	"net"
	"strconv"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestListenerPreflightDetectsBoundRawTCPReservationWithoutListen(t *testing.T) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if err := unix.Bind(fd, &unix.SockaddrInet4{Addr: [4]byte{127, 0, 0, 1}}); err != nil {
		t.Fatal(err)
	}
	bound, err := unix.Getsockname(fd)
	if err != nil {
		t.Fatal(err)
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(bound.(*unix.SockaddrInet4).Port))
	err = checkListenerClaims([]listenerClaim{{"tcp", address, "Auto candidate"}}, nil, bindListenerProbe)
	if !errors.Is(err, syscall.EADDRINUSE) {
		t.Fatalf("raw bound-only reservation was not detected: %v", err)
	}
}
