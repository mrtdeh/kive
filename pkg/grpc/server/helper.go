package grpc_server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/mrtdeh/kive/proto"

	"github.com/mrtdeh/kive/pkg/helper"
	"github.com/mrtdeh/kive/pkg/kive"

	Aany "github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// ========================== HEALTH CHECK =========================

func (a *agent) connHealthCheck(ctx context.Context, target *agent, d time.Duration) {
	msg := fmt.Sprintf("health check to %s", target.id)

	debug(a.id, "start %s", msg)
	defer func() {
		debug(a.id, "closed %s", msg)
	}()

	timer := time.NewTimer(d)

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.state.On(AgentStopping, AgentStopped):
			return
		case <-target.state.On(AgentStopping, AgentStopped):
			return
		case <-timer.C:
			if err := connIsFailed(target.conn); err != nil {
				target.err <- err
				return
			}
			_, err := target.proto.Ping(context.Background(), &proto.PingRequest{})
			if err != nil {
				target.err <- fmt.Errorf("error in ping request: %v", err)
				return
			}

			timer.Reset(d)
		}

	}

}

func connIsFailed(conn *grpc.ClientConn) error {
	status := conn.GetState()
	if status == connectivity.TransientFailure ||
		// status == connectivity.Idle ||
		status == connectivity.Shutdown {
		return fmt.Errorf("connection is failed with status %s", status)
	}
	return nil
}

func getRandHash(length int) string {
	b := rand.Intn(10000)
	b64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", b)))

	if len(b64) < length {
		return b64
	}

	return b64[:length]
}

// =============================================================
type DialConfig struct {
	Address     string
	DialContext func(context.Context, string) (net.Conn, error)
	Context     context.Context
}

func grpc_Dial(cnf DialConfig) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	var conn *grpc.ClientConn
	var err error

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if cnf.DialContext != nil {
		cnf.Address = ""
		opts = append(opts, grpc.WithContextDialer(cnf.DialContext))
	}
	if cnf.Context != nil {
		conn, err = grpc.DialContext(cnf.Context, cnf.Address, opts...)
	} else {
		conn, err = grpc.Dial(cnf.Address, opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("error in dial : %s", err.Error())
	}
	return conn, nil
}

func ConvertInterfaceToAny(v interface{}) (*Aany.Any, error) {
	anyValue := &Aany.Any{}
	bytes, _ := json.Marshal(v)
	bytesValue := &wrappers.BytesValue{
		Value: bytes,
	}
	err := anypb.MarshalFrom(anyValue, bytesValue, goproto.MarshalOptions{})
	return anyValue, err
}

func KiveRequest2ProtoKV(k kive.KVRequest) *proto.KV {
	return &proto.KV{
		Id:         k.Id,
		Key:        k.Key,
		Value:      k.Value,
		Action:     k.Action,
		Namespace:  k.Namespace,
		Timestamp:  k.Timestamp,
		DataCenter: k.DataCenter,
	}
}

func ProtoKV2KiveRequest(k *proto.KV) kive.KVRequest {
	return kive.KVRequest{
		Id:         k.Id,
		Key:        k.Key,
		Value:      k.Value,
		Action:     k.Action,
		Namespace:  k.Namespace,
		Timestamp:  k.Timestamp,
		DataCenter: k.DataCenter,
	}
}

func ProtoKVURequest(from, dc string, p []*proto.KVData_KVPartition) *proto.KVURequest {
	return &proto.KVURequest{
		Request: &proto.KVURequest_KvData{
			KvData: &proto.KVData{
				Partitions: p,
				From:       from,
				DC:         dc,
			},
		},
	}
}

func ProtoKVUErrorResponse(err error, rid string, code int32) *proto.KVUResponse {
	return &proto.KVUResponse{
		Response: &proto.KVUResponse_KvError{
			KvError: &proto.KVError{
				RecordId: rid,
				Error:    err.Error(),
				Code:     code,
			},
		},
	}
}

func KiveRequests2ProtoPartiotions(a *agent, items []kive.KVRequest, ignoreDC bool) ([]*proto.KVData_KVPartition, error) {
	col := make(map[string][]*proto.KV)
	for _, k := range items {
		if ignoreDC && !a.cluster.IsExistDC(k.DataCenter) {
			continue
		}
		if _, ok := col[k.DataCenter]; !ok {
			col[k.DataCenter] = []*proto.KV{}
		}

		col[k.DataCenter] = append(col[k.DataCenter], KiveRequest2ProtoKV(k))
	}

	if len(col) == 0 {
		return nil, nil
	}
	var partitions []*proto.KVData_KVPartition
	for dc, list := range col {

		t := kive.GetDCTimestamp(dc)
		partitions = append(partitions, &proto.KVData_KVPartition{
			Name:      dc,
			Records:   list,
			Timestamp: t,
		})
	}

	return partitions, nil
}

func KVDB2ProtoPartiotions(kvdb kive.KVDB, allowed_dcs ...string) ([]*proto.KVData_KVPartition, error) {
	var partitions []*proto.KVData_KVPartition
	for dc, d := range kvdb.Data {
		if len(allowed_dcs) > 0 && !helper.Contains(dc, allowed_dcs) {
			fmt.Println("debug allowed_dcs : ", allowed_dcs)
			continue
		}

		var r []*proto.KV
		for _, md := range d.Namespaces {
			for _, record := range md.Data {
				r = append(r, KiveRequest2ProtoKV(record))
			}
		}
		partitions = append(partitions, &proto.KVData_KVPartition{
			Name:      dc,
			Records:   r,
			Timestamp: d.Timestamp,
		})
	}

	if len(partitions) == 0 {
		return nil, nil
	}

	return partitions, nil
}
