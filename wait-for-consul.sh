#!/bin/sh
CONSUL_HOST=$1
shift
until curl -s $CONSUL_HOST/v1/status/leader | grep -q '\"'; do
  >&2 echo "Consul is unavailable - sleeping"
  sleep 5
done
>&2 echo "Consul is up - executing command"
exec "$@"