#!/bin/bash

DEBUG=1  WARN=1 ERROR=1 ./bin/centor -n hossain -dc dc2 -p 3002 -server -leader -primaries-addr "localhost:3000"
