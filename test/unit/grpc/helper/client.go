package grpc_test_helper

import (
	"context"
	"log"

	grpc_server "github.com/mrtdeh/kive/pkg/grpc/server"
)

func RunClientWithKive() (any, error) {

	a, err := grpc_server.NewServer(grpc_server.Config{
		Name:       "reza",
		DataCenter: "dc1",
		IsServer:   true,
		Servers:    []string{"localhost:3000"},
		Host:       "localhost",
		Port:       3001,
	})
	if err != nil {
		return nil, err
	}
	defer a.Stop()

	go func() {
		if err := a.Serve(nil); err != nil {
			log.Fatal(err)
		}
	}()

	a.GetCoreHandler().WaitForConnect(context.Background())
	return a, nil
}
