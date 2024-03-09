package routers

import (
	"io"

	api_v1 "github.com/mrtdeh/kive/routers/api/v1"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {

	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	pprof.Register(r)

	r.GET("/debug", api_v1.DebugInfo)
	r.GET("/call", api_v1.Call)
	r.GET("/nodes", api_v1.GetNodes)

	r.PUT("/kv", api_v1.PutKV)
	r.DELETE("/kv/:dc/:ns/*key", api_v1.DeleteKV)
	r.GET("/kv/:dc/:ns/*key", api_v1.GetKV)
	r.GET("/kv/:dc", api_v1.GetKV)

	return r
}
