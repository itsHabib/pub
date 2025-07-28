#!/bin/bash

set -euo pipefail
set -m

echo "Starting couchbase..."
/entrypoint.sh couchbase-server &

echo "waiting to reach couchbase..."
until curl -s http://localhost:8091/pools >/dev/null; do
  sleep 5
done

# check if cluster is already initialized
if ! couchbase-cli server-list -c localhost:8091 -u Administrator -p password >/dev/null; then

  # initialize cluster
  echo "Initializing cluster..."
  couchbase-cli cluster-init \
    --services data,index,query,fts \
    --index-storage-setting default \
    --cluster-ramsize 2048 \
    --cluster-index-ramsize 512 \
    --cluster-fts-ramsize 512 \
    --cluster-username Administrator \
    --cluster-password password \
    --cluster-name dockercompose

  echo "Creating pub bucket..."
  couchbase-cli bucket-create \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket "pubsub" \
    --bucket-type couchbase \
    --bucket-ramsize 512 \
    --wait

  echo "Creating pub bucket scope..."
  couchbase-cli bucket-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --scope create default

  echo "Creating pub bucket collection offset..."
  couchbase-cli collection-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --scope default \
    --collection create offset

  echo "Creating pub bucket collection cursor..."
  couchbase-cli collection-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --scope default \
    --collection create cursor

  echo "Creating pub bucket collection message..."
  couchbase-cli collection-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --scope default \
    --collection create message

  echo "Creating pub bucket collection lease..."
  couchbase-cli collection-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --scope default \
    --collection create lease
fi

fg 1
