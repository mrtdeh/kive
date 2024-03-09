package grpc_server

import (
	"context"
	"fmt"

	"github.com/mrtdeh/kive/proto"
)

func (a *agent) Call(ctx context.Context, req *proto.CallRequest) (*proto.CallResponse, error) {
	var tags []string
	tags = append(tags, a.id)

	if a.parent != nil && a.parent.id != req.AgentId {

		res, err := a.parent.proto.Call(context.Background(), &proto.CallRequest{
			AgentId: a.id,
		})
		if err != nil {
			return &proto.CallResponse{}, fmt.Errorf("failed to call parent %s: %v", a.parent.id, err)
		}
		tags = append(tags, res.Tags...)
	}

	if a.childs != nil {
		for _, c := range a.childs {
			if c.id != req.AgentId {
				status := c.connectivity.Value()
				if status != AgentConnected {
					Error(a.id, "call to %s ignored because of connection status is : %s\n", c.id, status)
					return &proto.CallResponse{}, fmt.Errorf("call ignore %s: %v", c.id, status)
					// continue
				}
				res, err := c.proto.Call(context.Background(), &proto.CallRequest{
					AgentId: a.id,
				})
				if err != nil {
					Error(a.id, "call to %s failed : %s\n", c.id, err)
					return &proto.CallResponse{}, fmt.Errorf("failed to call child %s: %v", c.id, err)
				}
				tags = append(tags, res.Tags...)
			}
		}
	}

	return &proto.CallResponse{Tags: tags}, nil
}
