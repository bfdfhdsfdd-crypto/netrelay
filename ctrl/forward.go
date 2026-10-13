package ctrl

import (
	"context"
	"net"

	"github.com/bfdfhdsfdd-crypto/netrelay/tunnel"
	log "github.com/kataras/golog"
)

type ForwardHandler struct {
	*tunnel.Client

	ctx context.Context
}

func NewForwardHandler(ctx context.Context, client *tunnel.Client) *ForwardHandler {
	return &ForwardHandler{
		Client: client,
	}
}

func (f *ForwardHandler) Handle(conn net.Conn) error {
	defer conn.Close()

	log.Infof("starting forwarding connection to: %s", f.Config.ForwardTarget)

	streamRW := tunnel.NewConn(f.ctx, f.Client)
	err := streamRW.ConnectMultiplex(f.Config.ForwardTarget)
	if err != nil {
		log.Warnf("failed to connect to target: %v", err)
		return err
	}

	log.Infof("connection established: %s", f.Config.ForwardTarget)

	f.DualPipe(conn, streamRW.ReadWriteCloser, f.Config.ForwardTarget)

	log.Infof("forward connection closed: %s", f.Config.ForwardTarget)
	return nil
}
