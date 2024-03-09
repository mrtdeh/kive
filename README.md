
# Kive
is a distributed and highly available key/value database.

## Features
- **Multi-Datacenter** : easy to build multi datacenter clusters and create stable connection to gather.


## Quick Start

You can simply build binary:
```sh
make build
```

And then print help for a more information :
```sh
./bin/centor -h
```
## Deploy and Test
1. Build dockerFile :
```sh
make docker-build
```
2. Running up the clusters mesh :
```sh
make docker-run
```
3. Test cluster discovery with curl :
```sh
# You can call the following API's from this endpoints:
# :9991 => dc1, :9992 => dc2, :9993 => dc3, :9994 => dc4

# Set new key/value pairs via API:
curl --location --request PUT 'http://localhost:9991/kv' \
--header 'Content-Type: application/json' \
--data '{
    "dc": "dc4",
    "ns": "foo", 
    "key":"bar",
    "value":"baz"
}'

# Get all keys from "foo" namespace:
curl --location 'http://localhost:9991/kv/dc4/foo'

# Get only specific key "bar":
curl --location 'http://localhost:9991/kv/dc4/foo/bar'

# Delete namespace "foo" with all keys:
curl --location --request DELETE 'http://localhost:9991/kv/dc4/foo'

# Delete key "bar":
curl --location --request DELETE 'http://localhost:9991/kv/dc4/foo/bar'

```

