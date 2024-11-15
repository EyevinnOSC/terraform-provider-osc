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

# Validate required variables are not empty
if [ -z "$ENCORE_URL" ] || [ -z "$EXTERNAL_ID" ] || [ -z "$EXTERNAL_BASENAME" ] || [ -z "$CALLBACK_URL" ]; then
  echo "Error: One or more Terraform outputs are missing. Ensure encore_url, name, and callback_url are set."
  exit 1
fi

echo "Encore URL: $ENCORE_URL"
echo "Encore Name: $EXTERNAL_ID"
echo "Segment BaseName: $EXTERNAL_BASENAME"
echo "Callback URL: $CALLBACK_URL"

TOKEN_URL="https://token.svc.$TF_VAR_osc_env.osaas.io/servicetoken"
ENCORE_TOKEN=$(curl -X 'POST' \
	$TOKEN_URL \
	-H 'Content-Type: application/json' \
	-H "x-pat-jwt: Bearer $TF_VAR_osc_pat"  \
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
