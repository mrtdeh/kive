package grpc_server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/mrtdeh/kive/proto"

	"github.com/mrtdeh/kive/pkg/kive"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type KVPS struct { // kv pools
	Items      []kive.KVRequest
	From       string
	DataCenter string
}
type KVManager struct {
	agent *agent
	pools []KVPS
	l     sync.RWMutex
	pl    sync.RWMutex
}

// create new instance of kvmanager and store it to agent
func (a *agent) setupKVManager() {
	if a.kvm != nil {
		return
	}
	// create new kvm object
	kvm := &KVManager{
		agent: a,
	}
	// store kvm object in the agent container
	a.kvm = kvm
	// run thread for sending kv in pool
	go kvm.poolKeeper()
}

// get agent kvm obejct out
func (a *agent) GetKVManager() *KVManager {
	// return back existed kvm
	return a.kvm

}

// this is a bridge between grpc server and kv client for sync data.
func (kvm *KVManager) Sync(prs []kive.KVRequest) {
	if prs != nil {
		go kvm.sync(prs, "", "")
	}
}

// =================================================================

// syncing local kvs requests to childs and parent server.
func (kvm *KVManager) sync(pr []kive.KVRequest, from, dc string) {
	if from == "" {
		from = kvm.agent.id
	}

	if dc == "" {
		dc = kvm.agent.dc
	}

	go kvm.sendKVtoAll(KVPS{
		Items:      pr,
		From:       from,
		DataCenter: dc,
	})
}

// add new kvs to manager pool
func (m *KVManager) addPool(kvps KVPS) {
	m.pl.Lock()
	defer m.pl.Unlock()

	m.pools = append(m.pools, kvps)
}

// poolKeeper function send stored kvs to parent if available
func (m *KVManager) poolKeeper() {
	for {

		<-m.agent.connectivity.On(AgentConnected)
		// stream send pool data to parent
		func() error {
			m.l.Lock()
			defer m.l.Unlock()
			// range in pools
			for j := 0; j < len(m.pools); j++ {
				// send pool's data to parent
				err := sendKVtoParent(m.agent, m.pools[j])
				if err != nil {
					log.Printf("error in send kv to parent in pool : %v", err)
					continue
				}
				// delete pool from array
				m.pools = append(m.pools[:j], m.pools[j+1:]...)

				// since we just deleted m.pools[j], we must redo that index
				j--
			}

			return nil
		}()

		time.Sleep(time.Second)
	}
}

// send kvs to parent and childs. if parent is disconnected then kvs store and try send later
func (kvm *KVManager) sendKVtoAll(kvps KVPS) error {
	kvm.l.Lock()
	defer kvm.l.Unlock()

	a := kvm.agent

	if a.isSubCluster {
		if a.parent != nil {
			tid := a.parent.id
			if tid != kvps.From {
				err := sendKVtoParent(a, kvps)
				if err != nil {
					fmt.Printf("error in send kvs to parent %s: %s", a.parent.id, err.Error())
					kvm.addPool(kvps)
					return err
				}
			}
		} else {
			kvm.addPool(kvps)
		}
	}
	if a.childs != nil {
		for _, c := range a.childs {
			tid := c.id
			if tid != kvps.From {
				fmt.Println("send kv to child : ", tid)
				err := sendKVtoChild(a, c, kvps)
				if err != nil {
					fmt.Printf("error in send kvs to child %s : %s", c.id, err.Error())
					continue
				}
			}
		}
	}

	return nil
}

// send kvs to childs
func sendKVtoChild(a *agent, c *agent, kvps KVPS) error {
	if c.connectivity.Value() != AgentConnected {
		return errors.New("status is not connected")
	}

	partitions, err := KiveRequests2ProtoPartiotions(a, kvps.Items, true)
	if err != nil {
		return err
	}

	str, err := c.proto.KVU(context.Background())
	if err != nil {
		return err
	}

	err = str.Send(ProtoKVURequest(a.id, kvps.DataCenter, partitions))
	if err != nil {
		return err
	}

	return str.CloseSend()
}

// send kvs to parent
func sendKVtoParent(a *agent, kvps KVPS) error {

	partitions, err := KiveRequests2ProtoPartiotions(a, kvps.Items, false)
	if err != nil {
		return err
	}

	str, err := a.parent.proto.KVU(context.Background())
	if err != nil {
		return err
	}
	err = str.Send(ProtoKVURequest(a.id, kvps.DataCenter, partitions))
	if err != nil {
		return err
	}

	return str.CloseSend()
}

// The main rpc function for request processing of kv data from parent and childs
func (a *agent) KVU(stream proto.Discovery_KVUServer) error {
	var ctx = context.Background()

	// Start a goroutine to receive messages from the gRPC stream
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == context.Canceled || err == io.EOF {
				debug(a.id, "canceled kvu connection : %v", err)
				return err
			}
			// check and return error when unavailable connection
			st, ok := status.FromError(err)
			if ok {
				if st.Code() == codes.Unavailable {
					debug(a.id, "unavailable kvu connection : %v", err)
					return err
				}
			}
			//  return any error else
			return fmt.Errorf("error in kvu recv : %s", err)
		}

		if data := req.GetKvData(); data != nil {
			var kvs []kive.KVRequest
			debug(a.id, "received kv data from %s", data.From)
			// check data timestamp
			for _, p := range data.Partitions {
				dcT := kive.GetDCTimestamp(p.Name)
				// check partition timestamp
				if p.Timestamp < dcT {
					continue
				}
				// update records in local database
				for _, r := range p.Records {
					// put kv record in local database
					if r.Action == "add" {
						_, err = kive.Put(r.DataCenter, r.Namespace, r.Key, r.Value, r.Timestamp)
						if err != nil {
							stream.Send(ProtoKVUErrorResponse(err, r.Id, 1))
							return err
						}
					}
					// delete record in local database
					if r.Action == "delete" {
						_, err = kive.Del(r.DataCenter, r.Namespace, r.Key, r.Timestamp)
						if err != nil {
							stream.Send(ProtoKVUErrorResponse(err, r.Id, 2))
							return err
						}
					}

					// append record to list for sending
					kvs = append(kvs, ProtoKV2KiveRequest(r))
				}

			}

			// sync kv records with other servers
			go a.kvm.sync(kvs, data.From, data.DC)
		}

		if t := req.GetKvTimestamps(); t != nil {

			if t.Timestamps != nil {

				var dcs []string
				for _, tt := range t.Timestamps {
					dcT := kive.GetDCTimestamp(tt.DC)
					if tt.Timestamp < dcT {
						dcs = append(dcs, tt.DC)
					}
				}

				if len(dcs) > 0 {
					if c := a.childs[t.From]; c != nil {
						if err := a.syncKVsDataMapWith(ctx, c, dcs...); err != nil {
							return err
						}
					}
				}

			}

		}

	}

}
