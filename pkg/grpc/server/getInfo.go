package grpc_server

import (
	"context"

	"github.com/mrtdeh/kive/proto"
)

func (s *agent) GetInfo(context.Context, *proto.EmptyRequest) (*proto.InfoResponse, error) {
	return &proto.InfoResponse{
		Id:       s.id,
		IsLeader: s.isLeader,
		Weight:   int32(s.weight),
	}, nil
}
