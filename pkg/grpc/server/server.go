package grpc_server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mrtdeh/kive/pkg/watcher"
)

type Config struct {
	Name       string   // Name of the server(id)
	DataCenter string   // Name of the server(id)
	Host       string   // Host of the server
	AltHost    string   // alternative host of the server (optional)
	Port       uint     // Port of the server
	Servers    []string // servers addresses for replication
	Primaries  []string // primaries addresses
	IsServer   bool     // is this node a server or not
	IsLeader   bool     // is this node leader or not
}

const (
	AgentStarting = "starting"
	AgentReady    = "ready"
	AgentStopping = "stopping"
	AgentStopped  = "stopped"

	AgentConnecting    = "connecting"
	AgentConnected     = "connected"
	AgentDisconnecting = "disconnecting"
	AgentDisconnected  = "disconnected"
)

func (a *agent) Stop() error {
	debug(a.id, "stopping agent")
	ctx := context.Background()
	s := a.state.Value()

	a.state.Set(AgentStopping)

	if s == AgentConnected {
		a.connectivity.WaitFor(ctx, AgentDisconnected)
	}

	// stoping server listener
	if a.grpcServer != nil {
		// send a close pulse to the connected clients
		if err := closeChilds(a); err != nil {
			return err
		}
		// TODO: wait for connections closed before stopping
		// time.Sleep(time.Millisecond * 100)

		// stoping grpc server
		a.grpcServer.Stop()
		debug(a.id, "closed grpc server")
	}
	a.state.Set(AgentStopped)
	debug(a.id, "agent stoped")
	return nil
}

func closeChilds(a *agent) error {
	a.cl.Lock()
	defer a.cl.Unlock()

	for _, child := range a.childs {
		err := func() error {
			if child.connectivity.Value() != AgentConnected {
				return nil
			}
			debug(a.id, "closing child %v", child.id)
			if err := child.conn.Close(); err != nil {
				return err
			}
			child.connectivity.Set(AgentDisconnected)
			debug(a.id, "closed child %v", child.id)

			return nil
		}()
		if err != nil {
			log.Println(err)
		}

	}
	return nil
}

func (a *agent) setupWatchers() {
	a.connectivity = watcher.NewWatcher("")
	a.state = watcher.NewWatcher("")
}

func NewServer(cnf Config) (*agent, error) {
	if cnf.Host == "" {
		cnf.Host = "127.0.0.1"
	}

	var servers []string
	// resolve alternative host from config
	var host string = cnf.Host
	if cnf.AltHost != "" {
		host = cnf.AltHost
	}
	// create default agent instance
	a := agent{
		id:       cnf.Name,
		dc:       cnf.DataCenter,
		addr:     fmt.Sprintf("%s:%d", host, cnf.Port),
		isServer: cnf.IsServer,
		isLeader: cnf.IsLeader,
		childs:   make(map[string]*agent),
	}
	a.setupClusterInfo()
	a.setupWatchers()
	a.setupKVManager()

	if !cnf.IsLeader || len(cnf.Primaries) > 0 {
		// if this node is a leader and no primaries are specified, this node becomes primary
		a.isSubCluster = true
	}

	ni := &NodeInfo{
		Id:         a.id,
		Address:    a.addr,
		IsServer:   a.isServer,
		IsLeader:   a.isLeader,
		DataCenter: a.dc,
	}
	// add current node info to nodes info map
	a.cluster.PutNode(ni, a.cluster.WithTag("start"))

	if !cnf.IsLeader && len(cnf.Servers) > 0 {
		servers = cnf.Servers
	} else {
		// if is a leader or there are no servers in the cluster

		if len(cnf.Primaries) > 0 {
			servers = cnf.Primaries
		}
	}

	if len(servers) > 0 {
		go func() {
			delay := time.Second * 1
			debug(a.id, "connector service started")
			timer := time.NewTimer(delay)
			a.state.WaitFor(context.Background(), AgentReady)
			for {
				select {
				case <-timer.C:
					debug(a.id, "try to connect to leader in the cluster")
					// try connect to parent server
					err := a.connect(servers)
					if err != nil {
						// set isConnected channel to false
						a.connectivity.Set(AgentDisconnected)
						// update cluster info
						if n, err := a.cluster.GetNode(a.id); err == nil {
							a.cluster.PutNode(n, a.cluster.WithTag("leave"))
						}
						Error(a.id, "error connecting to parent: %v", err)
					}
					timer.Reset(delay)
				case <-a.state.On(AgentStopping, AgentStopped):
					debug(a.id, "connector service stopped before agent is", a.state.Value())
					return
				}
			}
		}()
	}

	info(a.id, sPrintJson(cnf))

	return &a, nil
}
