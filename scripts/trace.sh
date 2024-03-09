#!/bin/bash

go test $1 -run $2 -trace=/tmp/tmptrace.out
go tool trace /tmp/tmptrace.out
