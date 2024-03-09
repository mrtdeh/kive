package grpc_test_helper

import (
	"context"
	"log"
	"time"

	"github.com/mrtdeh/kive/proto"

	grpc_server "github.com/mrtdeh/kive/pkg/grpc/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Server(ctx context.Context) (proto.DiscoveryClient, func()) {

	app, _ := grpc_server.NewServer(grpc_server.Config{
		Name:       "ali",
		DataCenter: "dc1",
		Host:       "localhost",
		Port:       3000,
		IsServer:   true,
		IsLeader:   true,
	})

	go func() {
		if err := app.Serve(nil); err != nil {
			log.Fatal(err)
		}
	}()

	app.GetCoreHandler().WaitForReady(ctx)

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

	conn, err := grpc.DialContext(queryCtx, "localhost:3000",

		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("error connecting to server: %v", err)
	}

	client := proto.NewDiscoveryClient(conn)

	closer := func() {
		cancel()
		app.Stop()
	}

	return client, closer
}

func ListenServer(ctx context.Context) func() error {

	app, _ := grpc_server.NewServer(grpc_server.Config{
		Name:       "ali",
		DataCenter: "dc1",
		Host:       "localhost",
		Port:       3000,
		IsServer:   true,
		IsLeader:   true,
	})

	go func() {
		if err := app.Serve(nil); err != nil {
			log.Fatal(err)
		}
	}()
	app.GetCoreHandler().WaitForReady(ctx)

	closer := func() error {
		return app.Stop()
	}

	return closer
}
