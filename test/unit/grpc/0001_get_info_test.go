package grpc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mrtdeh/kive/proto"

	grpc_test_helper "github.com/mrtdeh/kive/test/unit/grpc/helper"
)

func TestGetInfo(t *testing.T) {
	ctx := context.Background()

	client, closer := grpc_test_helper.Server(ctx)
	defer closer()

	type expectation struct {
		out *proto.InfoResponse
		err error
	}

	tests := map[string]struct {
		in       *proto.EmptyRequest
		expected expectation
	}{
		"Must_Success": {
			in: &proto.EmptyRequest{},
			expected: expectation{
				out: &proto.InfoResponse{
					Id: "ali",
				},
				err: nil,
			},
		},
	}

	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			out, err := client.GetInfo(ctx, tt.in)
			if err != nil {
				t.Error("getInfo error :", err)

			} else {
				if tt.expected.out.Id != out.Id {
					t.Errorf("Out -> \nWant: %v\nGot : %v", tt.expected.out, out)
				}

				fmt.Println("ok")
			}

		})
	}

}
