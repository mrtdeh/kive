package kive

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

const (
	dbPath = "/var/lib/centor/kive/db"
)

var db *KiveDB

type KiveServerInterface interface {
	Sync([]KVRequest)
}

type KVMapList struct {
	Data map[string]KVRequest `json:"data"`
	DC   string               `json:"dc"`
}

type NamespaceMapList map[string]KVMapList
type DataCenterMapList struct {
	Namespaces NamespaceMapList `json:"namespaces"`
	Name       string           `json:"name"`
	Timestamp  int64            `json:"timestamp"`
}
type KVDB struct {
	Data      map[string]DataCenterMapList
	Timestamp int64
}
type KiveDB struct {
	DataMap       KVDB `json:"db"`
	ServerHandler KiveServerInterface
	m             sync.RWMutex
}

type KVRequest struct {
	Id         string `json:"id"`
	Timestamp  int64  `json:"timestamp"`
	Key        string `json:"key"`
	Value      string `json:"value"`
	DataCenter string `json:"data_center"`
	Namespace  string `json:"namespace"`
	Action     string `json:"action"`
}

func Init(serverHandler KiveServerInterface) error {
	db = &KiveDB{
		DataMap: KVDB{
			Data: make(map[string]DataCenterMapList),
		},
		ServerHandler: serverHandler,
	}
	return nil
}

func LoadDB() error {
	data, err := os.ReadFile(dbPath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		log.Println("db is empty")
		return nil
	}

	err = json.Unmarshal(data, db)
	if err != nil {
		return err
	}

	return nil
}

func (db *KiveDB) updateTimestamp(dc string) {
	if m, ok := db.DataMap.Data[dc]; ok {
		m.Timestamp = time.Now().Unix()
		db.DataMap.Data[dc] = m
	}
}

func GetDCTimestamp(dc string) int64 {
	if t, ok := db.DataMap.Data[dc]; ok {
		return t.Timestamp
	}
	return 0
}

func GetTimestamp() int64 {
	return db.DataMap.Timestamp
}

func GetDataMap() KVDB {
	db.m.RLock()
	defer db.m.RUnlock()

	return db.DataMap
}

// TODO: replace KVRequest with Request(dc/ namespace / kv id / action)
func Del(dc, ns, key string, ts int64) (*KVRequest, error) {
	db.m.Lock()
	defer db.m.Unlock()
	var kv KVRequest
	// var namespaces NamespaceMapList
	var dcmap DataCenterMapList
	var m KVMapList
	var ok bool

	if dcmap, ok = db.DataMap.Data[dc]; !ok {
		return nil, fmt.Errorf("data center %s is not found", dc)
	}

	if m, ok = dcmap.Namespaces[ns]; !ok {
		return nil, fmt.Errorf("namespace %s is not found", ns)
	}

	if key != "" {
		// remove key
		if kv, ok = m.Data[key]; ok {
			currentTs := kv.Timestamp
			if ts < currentTs {
				return nil, fmt.Errorf("your opration is outdated : delete %s", key)
			}

			kv.Action = "delete"
			m.Data[key] = kv
		}
		delete(db.DataMap.Data[dc].Namespaces[ns].Data, key)
	} else {
		// remove namespace
		action := "delete"
		id := generateHash(action + key)
		kv = KVRequest{
			Id:         id,
			Timestamp:  time.Now().Unix(),
			Namespace:  ns,
			Action:     action,
			DataCenter: dc,
		}
		delete(db.DataMap.Data[dc].Namespaces, ns)
	}

	db.updateTimestamp(dc)
	return &kv, nil
}

// TODO: replace KVRequest with Request(dc/ namespace / kv id / action)
func Put(dc, ns, key, value string, ts int64) (*KVRequest, error) {
	db.m.Lock()
	defer db.m.Unlock()

	action := "add"
	id := generateHash(action + key)
	// var namespaces NamespaceMapList
	var dcmap DataCenterMapList
	var keys KVMapList
	var ok bool

	if dcmap, ok = db.DataMap.Data[dc]; !ok {
		dcmap = DataCenterMapList{
			Namespaces: make(NamespaceMapList),
			Name:       dc,
		}
		db.DataMap.Data[dc] = dcmap
	}

	if keys, ok = dcmap.Namespaces[ns]; !ok {
		keys = KVMapList{
			Data: map[string]KVRequest{},
			DC:   dc,
		}
		db.DataMap.Data[dc].Namespaces[ns] = keys
	}

	if kv, ok := keys.Data[key]; ok {
		currentTs := kv.Timestamp
		if ts < currentTs {
			return nil, fmt.Errorf("your opration is outdated : update %s", key)
		}
	}

	kv := KVRequest{
		Id:         id,
		Timestamp:  ts,
		Key:        key,
		Value:      value,
		Action:     action,
		Namespace:  ns,
		DataCenter: dc,
	}
	db.DataMap.Data[dc].Namespaces[ns].Data[key] = kv

	db.updateTimestamp(dc)
	return &kv, nil
}

func Sync(prs []KVRequest) error {
	if db.ServerHandler == nil {
		return fmt.Errorf("server handler is nil")
	}
	db.ServerHandler.Sync(prs)
	return nil
}

func Get(dc, ns, key string) (any, error) {
	db.m.RLock()
	defer db.m.RUnlock()
	// var n NamespaceMapList
	var dcmap DataCenterMapList
	var m KVMapList
	var ok bool

	// check data center is exist in database
	if dcmap, ok = db.DataMap.Data[dc]; !ok {
		return nil, fmt.Errorf("data center %s is not found", dc)
	}

	// return only dc info
	if ns == "" && key == "" {
		return map[string]interface{}{
			"timestamp": dcmap.Timestamp,
		}, nil
	}

	// check namespace is exist in database
	if m, ok = dcmap.Namespaces[ns]; !ok {
		return nil, fmt.Errorf("namespace %s is not found", ns)
	}
	// check if the key already exists return a value
	if key != "" {
		if kv, ok := m.Data[key]; ok {
			return kv, nil
		}
	} else {
		// if the key is not present then return the collection
		return m, nil
	}

	return nil, fmt.Errorf("key not found: %s", key)
}
