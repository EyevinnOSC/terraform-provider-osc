#!/bin/bash

# Ensure MEDIA_URL argument is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <MEDIA_URL>"
  exit 1
fi

# Assign the first argument to MEDIA_URL
MEDIA_URL="$1"

# Retrieve values from Terraform output
ENCORE_URL=$(terraform output -raw encore_url)
EXTERNAL_ID=$(terraform output -raw encore_name)
EXTERNAL_BASENAME=$(terraform output -raw encore_name)
CALLBACK_URL=$(terraform output -raw callback_url)
OSC_PAT=$TF_VAR_osc_pat
OSC_ENV=$TF_VAR_osc_env

# Validate required variables
if [ -z "$ENCORE_URL" ]; then
  echo "Error: Terraform output 'encore_url' is missing."
  exit 1
fi

if [ -z "$EXTERNAL_ID" ]; then
  echo "Error: Terraform output 'encore_name' is missing."
  exit 1
fi

if [ -z "$EXTERNAL_BASENAME" ]; then
  echo "Error: Terraform output 'encore_name' is missing."
  exit 1
fi

if [ -z "$CALLBACK_URL" ]; then
  echo "Error: Terraform output 'callback_url' is missing."
  exit 1
fi

if [ -z "$OSC_PAT" ]; then
  echo "Error: Environment variable 'OSC_PAT' (TF_VAR_osc_pat) is not set."
  exit 1
fi

if [ -z "$OSC_ENV" ]; then
  echo "Error: Environment variable 'OSC_ENV' (TF_VAR_osc_env) is not set."
  exit 1
fi

TOKEN_URL="https://token.svc.$OSC_ENV.osaas.io/servicetoken"
ENCORE_TOKEN=$(curl -X 'POST' \
    $TOKEN_URL \
    -H 'Content-Type: application/json' \
    -H "x-pat-jwt: Bearer $OSC_PAT"  \
    -d '{"serviceId": "encore"}' | jq -r '.token')

curl -X 'POST' \
  "$ENCORE_URL/encoreJobs" \
  -H "x-jwt: Bearer $ENCORE_TOKEN" \
  -H 'accept: application/hal+json' \
  -H 'Content-Type: application/json' \
  -d '{
  "externalId": "'"$EXTERNAL_ID"'",
  "profile": "program",
  "outputFolder": "/usercontent/",
  "baseName": "'"$EXTERNAL_BASENAME"'",
  "progressCallbackUri": "'"$CALLBACK_URL/encoreCallback"'",
  "inputs": [
    {
      "uri": "'"$MEDIA_URL"'",
      "seekTo": 0,
      "copyTs": true,
      "type": "AudioVideo"
    }
  ]
}'
