#!/bin/bash
while true; do
 make docker-down-all
 make docker-up
 ./scripts/test/test-stop-some-servers.sh
 ./scripts/test/test-stop-some-servers.sh
 sleep 1
 out=$(make test-integration)
 if [[ "$out" =~ "ok" ]]; then
  echo "OK"
 else
  echo "$out"
  exit 0
 fi

  # echo "Tests passed. Running again..."
done

echo "Tests failed. Exiting."
