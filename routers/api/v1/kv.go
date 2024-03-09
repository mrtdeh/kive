package api_v1

import (
	"strings"
	"time"

	"github.com/mrtdeh/kive/pkg/kive"

	"github.com/gin-gonic/gin"
)

type kvRequest struct {
	DataCenter string `json:"dc"`
	Namespace  string `json:"ns"`
	Key        string `json:"key"`
	Value      string `json:"value"`
}

func PutKV(c *gin.Context) {
	var kvr kvRequest
	if err := c.ShouldBind(&kvr); err != nil {
		c.JSON(400, gin.H{"error": "invalid input data"})
		return
	}

	// check access for request
	if !h.CanManageDC(kvr.DataCenter) {
		c.JSON(403, gin.H{"error": "access denied"})
		return
	}

	ts := time.Now().Unix()
	kv, err := kive.Put(kvr.DataCenter, kvr.Namespace, kvr.Key, kvr.Value, ts)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err := kive.Sync([]kive.KVRequest{*kv}); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "ok"})
}

func GetKV(c *gin.Context) {
	dc := c.Param("dc")
	// check access for request
	if !h.CanManageDC(dc) {
		c.JSON(403, gin.H{"error": "access denied"})
		return
	}

	ns := c.Param("ns")
	key := c.Param("key")
	key = strings.Trim(key, "/")

	res, err := kive.Get(dc, ns, key)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"result": res})
}

func DeleteKV(c *gin.Context) {
	dc := c.Param("dc")
	// check access for request
	if !h.CanManageDC(dc) {
		c.JSON(403, gin.H{"error": "access denied"})
		return
	}

	ns := c.Param("ns")
	key := c.Param("key")
	key = strings.Trim(key, "/")

	ts := time.Now().Unix()
	kv, err := kive.Del(dc, ns, key, ts)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if err := kive.Sync([]kive.KVRequest{*kv}); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "ok"})
}
