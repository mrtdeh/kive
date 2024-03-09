package grpc_server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mrtdeh/kive/proto"

	"github.com/mrtdeh/kive/pkg/watcher"

	"google.golang.org/grpc"
)

type eventBusInterface interface {
	Publish(string, ...any)
	Subscribe(string, any) error
}

type agent struct {
	// id of the agent
	id string
	// address of this node
	addr string
	// datacenter of this node
	dc string
	// is this node a server or not
	isServer bool
	// is this node leader or not
	isLeader bool
	// weight of this node in the cluster
	weight int
	// parent of this node in the cluster or in primary cluster
	parent *agent
	// childs map for store child nodes info
	childs map[string]*agent
	// childs locker
	cl sync.RWMutex
	// identify this server is a sub cluster of other or not
	isSubCluster bool
	// server listener
	listener net.Listener
	// grpc server
	grpcServer *grpc.Server
	// cluster info map
	cluster *ClusterInfo
	// kv database manager
	kvm *KVManager
	// client connection of requested server
	clientStream
	// watcher of service state
	state *watcher.Watcher
	// watcher of service connectivity
	connectivity *watcher.Watcher
}

type clientStream struct {
	conn  *grpc.ClientConn      // connection to the server
	proto proto.DiscoveryClient // discovery protocol
	err   chan error            // channel for error
}

func (a *agent) GetStreamError() <-chan error {
	return a.clientStream.err
}

func (a *agent) SetStreamError(err ...string) {
	a.clientStream.err <- errors.New(fmt.Sprint(err))
}

// ===========================================
type (
	NodeInfo struct {
		Id         string `json:"id" api:"id"`
		Name       string `json:"name" api:"name"`
		Address    string `json:"address" api:"address"`
		Port       string `json:"port" api:"port"`
		IsServer   bool   `json:"is_server" api:"is_server"`
		IsLeader   bool   `json:"is_leader" api:"is_leader"`
		ParentId   string `json:"parent_id" api:"parent_id"`
		DataCenter string `json:"data_center" api:"data_center"`
		StartedAt  int64  `json:"started_at" api:"started_at"` // started time in Microsecond
		JoinedAt   int64  `json:"joined_at" api:"joined_at"`   // joined time in Microsecond
		LeavedAt   int64  `json:"leaved_at" api:"leaved_at"`   // joined time in Microsecond
		ChangedAt  int64  `json:"changed_at"`                  // changed time in Microsecond
		Version    int64  `json:"version"`                     // version number
		Status     string `json:"status"`                      // client status, can be: Connected , Disconnected
	}
	NodesInfoMap map[string]NodeInfo
	MapList      map[string]struct{}
	ClusterInfo  struct {
		self  *agent
		l     *sync.RWMutex
		nodes NodesInfoMap
		dcs   MapList
	}
)

func (a *agent) setupClusterInfo() {
	a.cluster = &ClusterInfo{
		self:  a,
		nodes: make(map[string]NodeInfo),
		l:     &sync.RWMutex{},
		dcs:   make(MapList),
	}
}

type putOpt struct {
	IgnoreKeys        []string
	NoTimestampChange bool
	NoVersionChange   bool
	Tag               string
}
type PutOptions func(*putOpt)

func (c *ClusterInfo) renderOptions(options []PutOptions) *putOpt {
	var opt putOpt
	for _, o := range options {
		o(&opt)
	}
	return &opt
}

func (c *ClusterInfo) WithIgnoreKeys(ignoredKeys ...string) PutOptions {
	return func(o *putOpt) {
		o.IgnoreKeys = ignoredKeys
	}
}

func (c *ClusterInfo) WithNoTimestampChange() PutOptions {
	return func(o *putOpt) {
		o.NoTimestampChange = true
	}
}
func (c *ClusterInfo) WithNoVersionChange() PutOptions {
	return func(o *putOpt) {
		o.NoVersionChange = true
	}
}
func (c *ClusterInfo) WithTag(tag string) PutOptions {
	return func(o *putOpt) {
		o.Tag = tag
	}
}

// ================================================================

// func (n *NodeInfo) marshal() []byte {
// 	data, err := json.Marshal(*n)
// 	if err != nil {
// 		return nil
// 	}
// 	return data
// }

// ================================================================
func (c *ClusterInfo) updateDCs() {
	c.dcs = make(MapList)
	for _, n := range c.nodes {
		c.dcs[n.DataCenter] = struct{}{}
	}
}

func (c *ClusterInfo) IsExistDC(dc string) bool {
	c.l.Lock()
	defer c.l.Unlock()
	if _, ok := c.dcs[dc]; ok {
		return true
	}
	return false
}

func (c *ClusterInfo) DeleteNode(nodeId string) {
	c.l.Lock()
	defer c.l.Unlock()

	delete(c.nodes, nodeId)
	c.updateDCs()
}

func (c *ClusterInfo) PutNodeWithTimestamp(node *NodeInfo) {
	c.l.Lock()
	defer c.l.Unlock()

	node.ChangedAt = time.Now().UnixMicro()
	c.nodes[node.Id] = *node
	c.updateDCs()
}

// put node with tag(join/leave)
func (c *ClusterInfo) PutNode(node *NodeInfo, options ...PutOptions) {
	c.l.Lock()
	defer c.l.Unlock()

	opt := c.renderOptions(options)

	microtime := time.Now().UnixMicro()

	switch opt.Tag {
	case "start":
		node.StartedAt = microtime
		node.Status = "started"
	case "join":
		node.JoinedAt = microtime
		node.Status = "joined"
	case "leave":
		node.LeavedAt = microtime
		node.Status = "leaved"
	}

	if !opt.NoTimestampChange {
		node.ChangedAt = microtime
	}
	if !opt.NoVersionChange {
		node.Version++
	}

	c.nodes[node.Id] = *node

	c.updateDCs()
	debug(c.self.id, "put node %s with tag %s", node.Id, opt.Tag)
}

func (c *ClusterInfo) PutNodesMap(nodesMap NodesInfoMap) (changed bool) {
	c.l.Lock()
	defer c.l.Unlock()

	for k, node := range nodesMap {
		// check if the node is already exist in nodes list
		if n, ok := c.nodes[k]; ok {
			// if node version is newer than the current stored version then store the node info
			senaria1 := node.Version > n.Version
			senaria2 := node.Version == n.Version && node.ChangedAt > n.ChangedAt
			if senaria1 || senaria2 {
				// put item if version is greater than local version
				c.nodes[node.Id] = node
				changed = true
			}
		} else {
			// put if item not exists in nodes list
			c.nodes[node.Id] = node
			changed = true
		}
	}
	c.updateDCs()
	return changed

}

func (c *ClusterInfo) GetNode(nodeId string) (*NodeInfo, error) {
	c.l.RLock()
	defer c.l.RUnlock()
	if n, ok := c.nodes[nodeId]; ok {
		return &n, nil
	}
	return nil, fmt.Errorf("node id not found in cluster nodes map")
}

func (c *ClusterInfo) ToJson() (string, error) {
	c.l.Lock()
	defer c.l.Unlock()

	data, err := json.Marshal(c.nodes)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func agentToNodeInfo(a *agent) *NodeInfo {
	return &NodeInfo{
		Id:         a.id,
		Address:    a.addr,
		IsServer:   a.isServer,
		IsLeader:   a.isLeader,
		DataCenter: a.dc,
		ParentId:   a.parent.id,
	}
}
