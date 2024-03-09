package grpc_server

import (
	"context"

	"github.com/mrtdeh/kive/proto"
)

type CoreHandlers struct {
	agent *agent
}

func (a *agent) GetCoreHandler() *CoreHandlers {
	return &CoreHandlers{
		agent: a,
	}
}

// wait for current agent is running completely
func (h *CoreHandlers) WaitForReady(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.agent.state.On(AgentReady):
		return nil
	}
}

func (h *CoreHandlers) WaitForConnect(ctx context.Context) error {
	if !h.agent.isSubCluster {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.agent.connectivity.On(AgentConnected):
		return nil
	}
}

func (h *CoreHandlers) GetMyId() string {
	h.WaitForReady(context.Background())
	return h.agent.id
}

func (h *CoreHandlers) GetMyDC() string {
	h.WaitForReady(context.Background())
	return h.agent.dc
}

func (h *CoreHandlers) GetParentId() string {
	h.WaitForConnect(context.Background())
	if h.agent.parent != nil {
		return h.agent.parent.id
	}
	return ""
}

func (h *CoreHandlers) CanManageDC(dc string) bool {
	return h.agent.cluster.IsExistDC(dc)
}

// Todo: this function should be removed
func (h *CoreHandlers) Call(ctx context.Context) ([]string, error) {

	res, err := h.agent.Call(ctx, &proto.CallRequest{
		AgentId: h.agent.id,
	})
	if err != nil {
		return nil, err
	}
	return res.Tags, nil
}

// returns a map of all the nodes in the cluster
func (h *CoreHandlers) GetClusterNodes() map[string]NodeInfo {
	return h.agent.cluster.nodes
}
