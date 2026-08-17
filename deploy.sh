#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
AWS_REGION="${AWS_REGION:-eu-west-1}"

echo "=== Checking prerequisites ==="
for cmd in go terraform docker aws; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "ERROR: $cmd is required. Install it first." >&2
    exit 1
  fi
done

echo "=== Checking AWS credentials ==="
if ! aws sts get-caller-identity &>/dev/null; then
  echo "ERROR: Run 'aws configure' first." >&2
  exit 1
fi

AWS_ACCOUNT=$(aws sts get-caller-identity --query Account --output text)

if aws configure export-credentials --format env &>/dev/null; then
  eval "$(aws configure export-credentials --format env)"
fi

echo "Account: $AWS_ACCOUNT  Region: $AWS_REGION"

echo "=== Building presign-url Lambda ==="
cd presign-url
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
cd ..

echo "=== Building receipt-worker Lambda ==="
cd receipt-worker
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
cd ..

echo "=== Building csv-lister Lambda ==="
cd csv-lister
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
cd ..

echo "=== Building poppler-utils Lambda layer ==="
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

echo "=== Deployment complete ==="
ENDPOINT=$(terraform output -raw upload_endpoint 2>/dev/null || echo "")
echo "Upload endpoint: $ENDPOINT"
echo "Upload a receipt: curl -X POST \"$ENDPOINT?name=receipt.pdf\""
