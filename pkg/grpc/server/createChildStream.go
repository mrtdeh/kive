package grpc_server

import (
	"context"
	"time"

	"github.com/mrtdeh/kive/proto"
)

func (a *agent) CreateChildStream(ctx context.Context, c *agent) error {

	// dial to child listener
	conn, err := grpc_Dial(DialConfig{
		Address: c.addr,
		Context: ctx,
	})
	if err != nil {
		return err
	}
	defer conn.Close()
	// set client stream connection
	c.clientStream = clientStream{
		conn:  conn,
		proto: proto.NewDiscoveryClient(conn),
		err:   make(chan error, 1),
	}

	// done waiting for send connect back goroutine
	c.connectivity.Set(AgentConnected)
	// run health check conenction for this child
	go a.connHealthCheck(ctx, c, time.Second*5)

	// return back error message when child is disconnected or failed
	return <-c.GetStreamError()
}
