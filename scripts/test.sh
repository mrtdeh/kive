#!/bin/bash

# Sample list
my_list=("0001" "0002" "0003" "0004" "0005")

# Iterate over the list
for item in "${my_list[@]}"; do
    DEBUG=1 WARN=1 ERROR=1 go clean -testcache && go test -v -p 1 ./test/unit/grpc/${item}*

    # Add your logic here for each item
done


