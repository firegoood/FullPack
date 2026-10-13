package l3

import (
	"context"
	"net"

	"github.com/sirupsen/logrus"
)

// PacketDevice is the same bounded batched TUN contract the manual L3 engine
// uses. An enrolled coordinator can own it once across carrier changes.
type PacketDevice = packetDevice

// OpenPacketDevice validates the existing interface settings and preserves
// the manual engine's address, queue and MSS clamp setup/cleanup.
func OpenPacketDevice(cfg Config, log *logrus.Logger) (PacketDevice, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if log == nil {
		log = logrus.StandardLogger()
	}
	return openDevice(deviceSpec{Name: cfg.Iface, LocalIP: cfg.LocalIP, PeerIP: cfg.PeerIP, MTU: cfg.MTU, MSSClamp: cfg.MSSClamp, TxQueueLen: cfg.TxQueueLen, Qdisc: cfg.Qdisc, Log: log})
}

// OpenDatagramCarrier exposes the existing packet layer without a TUN or
// retransmission of user IP packets. The caller owns the returned carrier.
func OpenDatagramCarrier(ctx context.Context, cfg Config) (DatagramCarrier, net.Addr, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	return openCarrierContext(ctx, cfg)
}
