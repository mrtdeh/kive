package grpc_server

import (
	"context"
	"fmt"
	"time"

	"github.com/mrtdeh/kive/proto"
)

const (
	MaxApplyTries = 1024
)

// sending any status change of childs to parent server(if exist) and other children(if exists) connected to this server.
func (a *agent) syncClusterChanges(from string) error {
	// debug(a.id, "syncing cluster...")

	// get cluster map in json string
	data, err := a.cluster.ToJson()
	if err != nil {
		return err
	}

	// if has children then update children accordingly
	if len(a.childs) > 0 {
		// if a.isLeader {
		err = a.noticeNodesChangeToChilds(data, from)
		if err != nil {
			return fmt.Errorf("error in applyChange : %s", err.Error())
		}
		// debug(a.id, "sync cluster childs done")
	}

	// and if also is sub cluster then send change to top level cluster(primary cluster)
	if a.isSubCluster {
		var try int
		for {
			if try >= MaxApplyTries {
				return fmt.Errorf("max tries reached %d", MaxApplyTries)
			}
			try++

			if a.parent == nil {
				time.Sleep(time.Second * 1)
				continue
			}

			err := a.noticeNodesChangeTo(data, from, a.parent)
			if err != nil {
				Error(a.id, "error in sync Change To leader %s: %s", a.parent.id, err.Error())
				time.Sleep(time.Second * 1)
				continue
			}
			// debug(a.id, "sync cluster parent done")
			break
		}
	}
	// debug(a.id, "sync cluster done")
	return nil
}

func (a *agent) noticeNodesChangeTo(data, from string, to *agent) error {

	if to.id == from {
		return nil
	}

	// notice to parnet
	_, err := to.proto.Notice(context.Background(), &proto.NoticeRequest{
		From: a.id,
		Notice: &proto.NoticeRequest_NodesChange{
			NodesChange: &proto.NodesChange{
				Data:     data,
				DataType: proto.NodesChange_nodesMap,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("apply error: %v", err)
	}

	return nil
}

func (a *agent) noticeNodesChangeToChilds(data, from string) error {
	a.cl.RLock()
	defer a.cl.RUnlock()
	_ = from
	for _, child := range a.childs {
		// ignore disconnected child
		if child.connectivity.Value() != AgentConnected {
			Warn(a.id, "apply failed to %v ,status is %v", child.id, child.connectivity.Value())
			continue
		}
		// ignore leader child
		if child.isLeader {
			continue
		}

		// if child.id == from {
		// 	continue
		// }

		// notice to child
		_, err := child.proto.Notice(context.Background(), &proto.NoticeRequest{
			From: a.id,
			Notice: &proto.NoticeRequest_NodesChange{
				NodesChange: &proto.NodesChange{
					Data:     data,
					DataType: proto.NodesChange_nodesMap,
				},
			},
		})
		if err != nil {
			return fmt.Errorf("apply error: %v", err)
		}
	}

	return nil
}
