#!/bin/bash

go test -v ./test/grpc/connect_test.go | grep --color -E "DEBUG|ok"