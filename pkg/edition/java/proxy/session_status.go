package proxy

import (
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"

	"go.minekube.com/gate/pkg/edition/java/ping"
	"go.minekube.com/gate/pkg/edition/java/proto/packet"
	"go.minekube.com/gate/pkg/edition/java/proto/version"
	"go.minekube.com/gate/pkg/gate/proto"
)

type statusSessionHandler struct {
	conn    *minecraftConn
	inbound Inbound
	log     logr.Logger

	receivedRequest bool

	nopSessionHandler
}

func newStatusSessionHandler(conn *minecraftConn, inbound Inbound) sessionHandler {
	return &statusSessionHandler{conn: conn, inbound: inbound,
		log: conn.log.WithName("statusSession").WithValues(
			"inbound", inbound,
			"protocol", conn.protocol)}
}

func (h *statusSessionHandler) activated() {
	cfg := h.conn.proxy.Config()
	var log logr.Logger
	if cfg.Status.LogPingRequests || cfg.Debug {
		log = h.log
	} else {
		log = h.log.V(1)
	}
	log.Info("got server list status request")
}

func (h *statusSessionHandler) handlePacket(pc *proto.PacketContext) {
	if !pc.KnownPacket {
		// What even is going on? ;D
		_ = h.conn.close()
		return
	}

	switch p := pc.Packet.(type) {
	case *packet.StatusRequest:
		h.handleStatusRequest()
	case *packet.StatusPing:
		h.handleStatusPing(p)
	default:
		// unexpected packet, simply close
		_ = h.conn.close()
	}
}

var versionName = fmt.Sprintf("Gate %s", version.SupportedVersionsString)

func newInitialPing(p *Proxy, protocol proto.Protocol) *ping.ServerPing {
	shownVersion := protocol
	if !version.Protocol(protocol).Supported() {
		shownVersion = version.MaximumVersion.Protocol
	}
	return &ping.ServerPing{
		Version: ping.Version{
			Protocol: shownVersion,
			Name:     versionName,
		},
		Players: &ping.Players{
			Online: p.PlayerCount(),
			Max:    p.config.Status.ShowMaxPlayers,
		},
		Description: p.motd,
		Favicon:     p.favicon,
	}
}

func (h *statusSessionHandler) handleStatusRequest() {
	if h.receivedRequest {
		// Already sent response
		_ = h.conn.close()
		return
	}
	h.receivedRequest = true

	e := &PingEvent{
		inbound: h.inbound,
		ping:    newInitialPing(h.proxy(), h.conn.protocol),
	}
	h.proxy().event.Fire(e)

	if e.ping == nil {
		_ = h.conn.close()
		h.log.V(1).Info("ping response was set to nil by an event handler, no response is sent")
		return
	}
	if !h.inbound.Active() {
		return
	}

	response, err := json.Marshal(e.ping)
	if err != nil {
		_ = h.conn.close()
		h.log.Error(err, "error marshaling ping response to json")
		return
	}
	_ = h.conn.WritePacket(&packet.StatusResponse{
		Status: string(response),
	})
}

func (h *statusSessionHandler) handleStatusPing(p *packet.StatusPing) {
	// Just return again and close
	defer h.conn.close()
	if err := h.conn.WritePacket(p); err != nil {
		h.log.V(1).Info("error writing StatusPing response", "err", err)
	}
}

func (h *statusSessionHandler) proxy() *Proxy {
	return h.conn.proxy
}
