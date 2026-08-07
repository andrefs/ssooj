#!/usr/bin/env bash
# Full deployment: build binaries, build layer, apply Terraform.
set -euo pipefail

cd "$(dirname "$0")"

AWS_ACCOUNT="${AWS_ACCOUNT:-ACCOUNT_ID}"
AWS_REGION="${AWS_REGION:-eu-west-1}"

echo "=== Building presign-url Lambda ==="
cd presign-url
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
cd ..

echo "=== Building receipt-worker Lambda ==="
cd receipt-worker
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
cd ..

echo "=== Building poppler-utils layer ==="
cd infrastructure
bash build-layer.sh
mkdir -p artifacts
cd ..

echo "=== Applying Terraform ==="
cd infrastructure
terraform init -upgrade
terraform apply \
  -var="account_id=$AWS_ACCOUNT" \
  -var="aws_region=$AWS_REGION" \
  -auto-approve

echo "=== Done ==="
terraform output
