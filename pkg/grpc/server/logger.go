package grpc_server

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func glog(level, meta, format string, a ...any) {
	if os.Getenv(level) != "" {
		if meta == "" {
			meta = getRandHash(5)
		}
		timestamp := fmt.Sprint(time.Now().UnixMicro())
		timestamp = timestamp[7:]
		fmt.Printf("["+timestamp+"]["+level+"]:: "+format+" ::["+meta+"]\n", a...)
	}
}

func debug(meta, format string, a ...any) {
	glog("DEBUG", meta, format, a...)
}

func info(meta, format string, a ...any) {
	glog("INFO", meta, format, a...)
}

func Error(meta, format string, a ...any) {
	glog("ERROR", meta, format, a...)
}

func Warn(meta, format string, a ...any) {
	glog("WARN", meta, format, a...)
}

func sPrintJson(obj interface{}) string {
	j, err := json.MarshalIndent(obj, "", " ")
	if err != nil {
		panic("error marshalling : " + err.Error())
	}
	return string(j)
}
