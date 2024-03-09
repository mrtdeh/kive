package grpc_server

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/mrtdeh/kive/proto"

	"github.com/mrtdeh/kive/pkg/kive"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// connect method used to find leader server and connect to it.
func (a *agent) connect(addrs []string) error {
	a.connectivity.Set(AgentConnecting)

	if len(addrs) == 0 {
		return nil
	}
	servers := addrs

	// master election for servers / best election for clients
	var si *ServerInfo
	var err error
	if a.isServer {
		// select leader only
		si, err = leaderElect(servers)
	} else {
		// select best server in server's pool
		si, err = bestElect(servers)
	}
	if err != nil {
		return err
	}

	debug(a.id, "connecting to %s", si.Id)
	// dial to selected server
	conn, err := grpc_Dial(DialConfig{
		Address: si.Addr,
	})
	if err != nil {
		return err
	}
	defer conn.Close()
	// create parent object
	a.parent = &agent{
		// parent agent
		addr:     si.Addr,
		id:       si.Id,
		isLeader: si.IsLeader,
		clientStream: clientStream{ // parent stream
			conn:  conn,
			proto: proto.NewDiscoveryClient(conn),
			err:   make(chan error, 1),
		},
	}
	a.parent.setupWatchers()

	if n, err := a.cluster.GetNode(a.id); err == nil {
		n.ParentId = si.Id
		a.cluster.PutNode(n, a.cluster.WithTag("join"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// create sync stream rpc to parent server
	go func() {
		err = a.grpc_connect(ctx, a.parent)
		if err != nil {
			a.parent.SetStreamError("error in grpc_conenct : ", err.Error())
		}
	}()

	a.connectivity.WaitFor(ctx, AgentConnected)
	// send current cluster nodes to parent server
	if err := a.syncClusterInfoWith(ctx, a.parent); err != nil {
		return fmt.Errorf("error in sync cluster info : %s", err.Error())
	}
	// send current cluster KVs data to parent server
	if err := a.syncKVsDataMapWith(ctx, a.parent); err != nil {
		return fmt.Errorf("error in sync KVs data : %s", err.Error())
	}
	// send current/sub dc's timestamp to parent server
	if err := a.syncKVTimestampsToParent(ctx); err != nil {
		return fmt.Errorf("error in sync KVs data : %s", err.Error())
	}

	// health check conenction for parent server
	go a.connHealthCheck(ctx, a.parent, time.Second*5)

	return <-a.parent.GetStreamError()
}

func (a *agent) syncKVTimestampsToParent(ctx context.Context) error {
	dm := kive.GetDataMap()

	str, err := a.parent.proto.KVU(ctx)
	if err != nil {
		return err
	}

	var dct []*proto.KVTimestamps_DCTimestamp
	for _, d := range dm.Data {
		dct = append(dct, &proto.KVTimestamps_DCTimestamp{
			DC:        d.Name,
			Timestamp: d.Timestamp,
		})
	}
	err = str.Send(&proto.KVURequest{
		Request: &proto.KVURequest_KvTimestamps{
			KvTimestamps: &proto.KVTimestamps{
				Timestamps: dct,
			},
		}})
	if err != nil {
		return err
	}

	return str.CloseSend()
}

func (a *agent) syncKVsDataMapWith(ctx context.Context, target *agent, allowed_dcs ...string) error {
	dm := kive.GetDataMap()

	partitions, err := KVDB2ProtoPartiotions(dm, allowed_dcs...)
	if err != nil {
		return err
	}

	str, err := target.proto.KVU(ctx)
	if err != nil {
		return err
	}

	if partitions == nil {
		return nil
	}

	err = str.Send(ProtoKVURequest(a.id, a.dc, partitions))
	if err != nil {
		return err
	}

	return str.CloseSend()
}

func (a *agent) syncClusterInfoWith(ctx context.Context, target *agent) error {
	// debug(a.id, "syncing nodes data...")
	jsonMap, err := a.cluster.ToJson()
	if err != nil {
		return err
	}

	_, err = target.proto.Notice(ctx, &proto.NoticeRequest{
		From: a.id,
		Notice: &proto.NoticeRequest_NodesChange{
			NodesChange: &proto.NodesChange{
				Data:     jsonMap,
				DataType: proto.NodesChange_nodesMap,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error in call Change rpc : %s", err.Error())
	}

	// debug(a.id, "sync nodes data to parent(%s", a.parent.id)

	return nil
}

// grpc_connect function used to call Connect rpc to parent server in bi-directional.
func (a *agent) grpc_connect(ctx context.Context, target *agent) error {

	stream, err := target.proto.Connect(ctx)
	if err != nil {
		return fmt.Errorf("error in create connect stream : %s", err.Error())
	}
	// send connect message to parent server
	err = stream.Send(&proto.ConnectMessage{
		Msg: &proto.ConnectMessage_Info_{
			Info: &proto.ConnectMessage_Info{
				Id:         a.id,
				DataCenter: a.dc,
				Addr:       a.addr,
				IsServer:   a.isServer,
				IsLeader:   a.isLeader,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error in send connect message : %s", err.Error())
	}

	// assing parent stream handler to agent object

	messageCh := make(chan *proto.ConnectBackMessage)
	errorCh := make(chan error)
	// Start a goroutine to receive messages from the gRPC stream
	go func() {
		for {
			res, err := stream.Recv()
			if err != nil {
				errorCh <- err
				return
			}
			messageCh <- res
		}
	}()

	// pid: parent id
	var pid string
	for {

		select {

		// # receive connect message from parent server
		case res := <-messageCh:
			if res != nil {
				pid = res.Id
			}
			debug(a.id, "conenct back from parent - ID=%s", pid)
			// set isConnected to true
			a.connectivity.Set(AgentConnected)

		// # return on context cancellation
		case <-ctx.Done():
			a.connectivity.Set(AgentDisconnected)
			return fmt.Errorf("context canceled : %s", ctx.Err())

		case <-a.state.On(AgentStopping, AgentStopped):
			a.connectivity.Set(AgentDisconnected)
			return fmt.Errorf("agent stopping")

		// # return in case of error
		case err := <-errorCh:
			if err != nil {
				a.connectivity.Set(AgentDisconnected)
				// defer debug message
				defer func() {
					if pid != "" {
						debug(a.id, "disconnect from parent - ID=%s", pid)
					}
				}()

				// return error on server error is context cancellation
				if err == context.Canceled || err == io.EOF {
					debug(a.id, "canceled parent connection : %v", target.id)
					return err
				}

				// check and return error when unavailable connection
				st, ok := status.FromError(err)
				if ok {
					if st.Code() == codes.Unavailable {
						debug(a.id, "unavailable parent connection : %v", target.id)
						return err
					}
				}

				//  return any error else
				Error(a.id, "error in recv : %v", err)
				return fmt.Errorf("error in recv : %s", err)
			}
		}

	}

}
