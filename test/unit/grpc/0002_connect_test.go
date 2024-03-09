package grpc_test

import (
	"context"
	"log"
	"sync"
	"testing"

	"github.com/mrtdeh/kive/proto"

	grpc_server "github.com/mrtdeh/kive/pkg/grpc/server"
	grpc_test_helper "github.com/mrtdeh/kive/test/unit/grpc/helper"
)

func TestConnect(t *testing.T) {
	ctx := context.Background()

	closer := grpc_test_helper.ListenServer(ctx)
	defer closer()

	type expectation struct {
		out *proto.InfoResponse
		err error
	}

	tests := map[string]struct {
		in       *grpc_server.Config
		expected expectation
	}{
		"Must_Success": {
			in: &grpc_server.Config{
				Name:       "reza",
				DataCenter: "dc1",
				IsServer:   true,
				Servers:    []string{"localhost:3000"},
				Host:       "localhost",
				Port:       3001,
			},
			expected: expectation{
				out: &proto.InfoResponse{
					Id: "ali",
				},
				err: nil,
			},
		},
	}

	var wg sync.WaitGroup
	for scenario, tt := range tests {
		wg.Add(1)
		t.Run(scenario, func(t *testing.T) {
			go func() {
				defer wg.Done()

				a, err := grpc_server.NewServer(*tt.in)
				if err != nil {
					t.Errorf("Err -> %s\n", err)
				}
				defer a.Stop()

				go func() {
					if err := a.Serve(nil); err != nil {
						log.Fatal(err)
					}
				}()

				a.GetCoreHandler().WaitForConnect(ctx)

				res, err := a.Call(ctx, &proto.CallRequest{
					AgentId: a.GetCoreHandler().GetMyId(),
				})
				if err != nil {
					t.Errorf("Err -> \nGot: %q\n", err)
				} else {
					if len(res.Tags) != 2 {
						t.Errorf("tags must ne 2 but got %d\n", len(res.Tags))
					}
				}
			}()

		})
	}
	wg.Wait()
}
