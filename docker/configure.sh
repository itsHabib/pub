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
  couchbase-cli collection-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --create-scope default

  echo "Creating pub bucket collection offsets..."
  couchbase-cli collection-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --create-collection default.offsets

  echo "Creating pub bucket collection cursors..."
  couchbase-cli collection-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --create-collection default.cursors

  echo "Creating pub bucket collection messages..."
  couchbase-cli collection-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --create-collection default.messages

  echo "Creating pub bucket collection leases..."
  couchbase-cli collection-manage \
    --cluster localhost \
    --username Administrator \
    --password password \
    --bucket pubsub \
    --create-collection default.leases

  echo "Waiting for collections to be ready..."
  sleep 10

  echo "Creating performance index for messages..."
  cbq -e http://localhost:8093 -u Administrator -p password -s="
    CREATE INDEX idx_messages_topic_shard_offset 
    ON \`pubsub\`.\`default\`.\`messages\`(topic, shard, \`offset\`)
    WHERE topic IS NOT MISSING;
  "
fi

fg 1
