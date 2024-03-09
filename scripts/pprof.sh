#!/bin/bash

# curl http://localhost:9991/debug/pprof/heap > /tmp/heap.out 
# go tool pprof -http=:9090 /tmp/heap.out

go tool  pprof -http=:9090 http://localhost:9991/debug/pprof/heap

