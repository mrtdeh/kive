package api_v1

import (
	"fmt"
	"runtime"

	"github.com/gin-gonic/gin"
)

func DebugInfo(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c.JSON(200, gin.H{
		"memory": fmt.Sprintf("%d MB", m.Alloc/(1024*1024)),
	})
}
