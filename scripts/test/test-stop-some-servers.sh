#!/bin/bash

docker stop dc1-centor-server-1-1-1

docker stop dc2-centor-server-2-1-1

docker stop dc4-centor-server-4-1-1

docker stop dc4-centor-client-4-1-1

docker stop dc3-centor-client-3-1-1


sleep 2

docker start dc3-centor-client-3-1-1

docker start dc4-centor-client-4-1-1

docker start dc4-centor-server-4-1-1

docker start dc2-centor-server-2-1-1

docker start dc1-centor-server-1-1-1





