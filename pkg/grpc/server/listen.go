package grpc_server

import (
	"fmt"
	"net"
	"time"

	"github.com/mrtdeh/kive/proto"

	"google.golang.org/grpc"
)

const (
	MaxListenTries = 10
	ListenTryDelay = 100
)

func (a *agent) Serve(lis net.Listener) error {
	grpcServer := grpc.NewServer()
	if lis == nil {
		var err error
		try := 0
		for {
			if try >= MaxListenTries {
				return fmt.Errorf("error creating the server %v", err)
			}
			try++

			debug(a.id, "trying to listen on %v", a.addr)

			lis, err = net.Listen("tcp", a.addr)
			if err != nil {
				fmt.Println(err)
				time.Sleep(time.Millisecond * ListenTryDelay)
				continue
			}
			break
		}
	}
	proto.RegisterDiscoveryServer(grpcServer, a)

	a.listener = lis
	a.grpcServer = grpcServer
	a.state.Set(AgentReady)

	debug(a.id, "listen an %s", a.addr)
	return grpcServer.Serve(lis)
}
