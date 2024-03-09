#!/bin/bash

if [ -z "$1" ]; then
  echo "Usage: $0 /path/to/directory"
  exit 1
fi

root_path="$1"

echo $(readlink -m $root_path):
echo

# Function to count lines for a given directory
count_lines() {
  local dir=$1
  local bdir=${2:-"$(basename $dir)"}
  local count=$(find "$dir" -type f -name "*.go" ! -name "*_test.go" ! -name "*.pb.go"  ! -name "docs.go" -exec cat {} \; | wc -l)
  if [ "$count" -gt 0 ]; then
    echo $bdir : $count
  fi
}
# Count lines for the root path
count_lines "$root_path" ./

# Find all directories in the root path (including subdirectories)
directories=$(find "$root_path" -maxdepth 1 -type d)
# Loop through each directory
for dir in $directories; do
  # Check if the directory is not the root path
  if [ "$dir" != "$root_path" ]; then
    count_lines "$dir"
  fi
done | sort -t':' -k2 -n -r


