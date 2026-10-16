package peerauto

import smux "github.com/backpack/backpack/internal/tunnel/peerauto/flowmux"

// MuxFlowStatus describes local transport behavior, never enrollment policy.
// Old peers retain v1 and the bounded shared bucket until both are upgraded.
type MuxFlowStatus struct {
	BufferedReceive   int64 `json:"bufferedReceive"`
	Version           int   `json:"version"`
	Isolated          bool  `json:"isolated"`
	MaxBufferedStream int   `json:"maxBufferedStream"`
}

func muxSession(s carrierSession) *smux.Session {
	switch c := s.(type) {
	case noiseCarrier:
		return c.s
	case boundCarrier:
		return muxSession(c.carrierSession)
	case *helperCarrier:
		return muxSession(c.carrierSession)
	case *rawPacketCarrier:
		return muxSession(c.carrierSession)
	default:
		return nil // native Wat and QUIC retain their own flow control
	}
}

func enableAuthenticatedFlowControl(s carrierSession) error {
	if mux := muxSession(s); mux != nil {
		version, _, _ := mux.FlowControl()
		if version == 2 {
			return mux.EnableAuthenticatedFlowControl()
		}
	}
	return nil
}

func muxFlowStatus(s carrierSession) *MuxFlowStatus {
	if mux := muxSession(s); mux != nil {
		version, isolated, window := mux.FlowControl()
		return &MuxFlowStatus{BufferedReceive: mux.BufferedBytes(), Version: version, Isolated: isolated, MaxBufferedStream: window}
	}
	return nil
}
