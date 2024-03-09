package grpc_server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mrtdeh/kive/proto"
)

func (a *agent) Notice(ctx context.Context, req *proto.NoticeRequest) (*proto.Close, error) {

	c := &proto.Close{}
	// debug(a.id, "received notice from %s", req.From)

	if nch := req.GetNodesChange(); nch != nil {

		// check for change type
		switch nch.DataType {
		// trigger on data from leader or server
		case proto.NodesChange_nodesMap:
			// parsing recieved nodes map of cluster
			if err := NodesInfoMapParser(a, nch, req.From); err != nil {
				return c, err
			}

		default:
			return c, fmt.Errorf("data type not supported : %v", nch.DataType)

		}

	}

	return &proto.Close{}, nil
}

func NodesInfoMapParser(a *agent, nch *proto.NodesChange, from string) error {
	var nim NodesInfoMap
	err := json.Unmarshal([]byte(nch.Data), &nim)
	if err != nil {
		return err
	}

	// update the node info map
	a.cluster.PutNodesMap(nim)
	// remove disconnected childs
	a.removeDisconnectedChilds()
	// sync nodes data
	err = a.syncClusterChanges(from)
	if err != nil {
		return err
	}

	return nil
}

func (a *agent) removeDisconnectedChilds() {
	a.cl.Lock()
	defer a.cl.Unlock()

	var keysToRemove []string

	for key, c := range a.childs {
		if n, _ := a.cluster.GetNode(c.id); n != nil {
			if n.ParentId != a.id {
				debug(a.id, "removing child %s beacuse of parent is %s", key, n.ParentId)
				keysToRemove = append(keysToRemove, key)
			}
		}
	}

	for _, key := range keysToRemove {
		debug(a.id, "removed child %s", key)
		delete(a.childs, key)
	}
}
