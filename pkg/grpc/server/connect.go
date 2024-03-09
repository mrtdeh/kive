package grpc_server

import (
	"context"
	"fmt"
	"io"

	"github.com/mrtdeh/kive/proto"
)

const (
	StatusDisconnected = "disconnected"
	StatusConnected    = "connected"
	StatusConnecting   = "connecting"
)

func (a *agent) Connect(stream proto.Discovery_ConnectServer) error {
	var c *agent
	var messageCh = make(chan *proto.ConnectMessage, 1)
	var errorCh = make(chan error, 1)

	var ctx = context.Background()

	// receive message from in sperated goroutine
	go func() {
		for {
			// receive connect message from child server
			res, err := stream.Recv()
			if err != nil {
				errorCh <- err
				break
			}
			messageCh <- res
		}
	}()
	// receive connect message from channel
	for {
		select {

		// wait for receive message
		case res := <-messageCh:
			if info := res.GetInfo(); info != nil {
				// create new child agent object and fill with gained information
				c = &agent{
					id:       info.Id,
					dc:       info.DataCenter,
					addr:     info.Addr,
					isServer: info.IsServer,
					isLeader: info.IsLeader,
					parent: &agent{
						id: a.id,
					},
				}
				c.setupWatchers()

				ni := agentToNodeInfo(c)
				// store client connection
				err := a.addchild(c) // add child
				if err != nil {
					return err
				}

				a.cluster.PutNode(ni,
					a.cluster.WithNoTimestampChange(),
					a.cluster.WithNoVersionChange(),
					a.cluster.WithTag("join"),
				)

				defer func() error {
					a.cluster.PutNode(ni,
						a.cluster.WithNoTimestampChange(),
						a.cluster.WithNoVersionChange(),
						a.cluster.WithTag("leave"),
					)
					// leave child from joined server
					if err := a.leavechild(c.id); err != nil {
						Error(a.id, "leave child %v error : %v", c.id, err)
						return err
					}
					c.SetStreamError("client disconnected")

					// send change for remove client to leader
					err := a.syncClusterChanges(a.id)
					if err != nil {
						Error(a.id, "sync error : %v", err)
						return fmt.Errorf("error in sync change : %s", err.Error())
					}
					return nil
				}()

				// Dial back to joined server
				go func() {
					err := a.CreateChildStream(ctx, c)
					if err != nil {
						errorCh <- fmt.Errorf("error in create child stream : %s", err.Error())
					}
				}()

				// Send child status to leader
				go func() {
					// wait for child to connect done
					c.connectivity.WaitFor(ctx, AgentConnected)
					// send back agent id to connected client
					err = stream.Send(&proto.ConnectBackMessage{Id: a.id})
					if err != nil {
						errorCh <- fmt.Errorf("error in connect back message: %v", err)
					}
					// debug(a.id, "sending back message to connected client")
					// then, send changes to leader
					err := a.syncClusterChanges(a.id)
					if err != nil {
						errorCh <- fmt.Errorf("error in sync change : %s", err.Error())
					}
				}()
			}

		// wait for error
		case err := <-errorCh:

			if err == context.Canceled || err == io.EOF {
				debug(a.id, "canceled child connection : %v", c.id)
				return nil
			}

			Error(a.id, "conenction failed for %s : %s", c.id, err.Error())
			return fmt.Errorf("error in recv: %v", err)

		} // end select
	} // end for
}

func (a *agent) leavechild(id string) error {
	a.cl.Lock()
	defer a.cl.Unlock()
	var c *agent

	if c = a.childs[id]; c == nil {
		return fmt.Errorf("this join id is not exist for leaving : %s", id)
	}
	if c.connectivity.Value() == AgentDisconnected {
		return nil
	}
	c.connectivity.Set(AgentDisconnected)
	a.weight--

	c.connectivity.Set(AgentDisconnected)
	c.state.Set(AgentStopping)
	debug(a.id, "disconnect client - ID=%s", id)

	return nil
}
func (a *agent) addchild(c *agent) error {
	a.cl.Lock()
	defer a.cl.Unlock()
	if cc, exist := a.childs[c.id]; exist && cc.connectivity.Value() == AgentConnected {
		return fmt.Errorf("this join id already exist : %s", c.id)
	}
	// add requested to childs list
	c.connectivity.Set(AgentConnecting)
	a.childs[c.id] = c
	a.weight++

	c.state.Set(AgentStarting)
	debug(a.id, "added new client - ID=%s", c.id)
	return nil
}
