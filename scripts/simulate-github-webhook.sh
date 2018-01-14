#!/usr/bin/env bash

url=$1
curl -v -H "Content-Type: application/json" -X POST -d @scripts/sample-github-payload-CONTRIBUTOR.json http://${url}/trigger
