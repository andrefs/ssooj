# ssooj -- Supermarket Receipt PDF Parser

Extracts structured data from supermarket receipt PDFs and writes
it to CSV (S3) and DynamoDB.

Built with Go, AWS Lambda, SQS, DynamoDB, Terraform.

## Supported Chains

| Chain | Status |
|---|---|
| Continente (Portugal) | Supported |

The parser is extensible via a registration pattern. To add a new chain,
create a file in `receipt-worker/receipt/` implementing the `Parser` interface
and register it in an `init()` function.

## Architecture

```
Upload PDF --[API Gateway]--> Presign URL Lambda (Go)
                                    |
                          User uploads PDF directly to S3
                                    |
                          S3 event -> SQS -> Worker Lambda (Go + pdftotext)
                                    |
                          CSV in S3 + DynamoDB item
```

- **Presign URL Lambda** (`presign-url/`): Returns a time-limited presigned PUT
  URL so the user can upload a PDF directly to S3 without AWS credentials.
- **Worker Lambda** (`receipt-worker/`): Triggered by SQS when a new PDF lands
  in the raw bucket. Downloads the PDF, runs `pdftotext -layout`, parses the
  receipt text, and writes a CSV row per item to S3 plus a full receipt item
  to DynamoDB.
- **Infrastructure** (`infrastructure/`): Terraform definitions for S3, SQS,
  DynamoDB, IAM, Lambda functions, API Gateway, and the poppler-utils layer.

## Prerequisites

- [Go](https://go.dev/dl/) >= 1.25
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.7
- [Docker](https://docs.docker.com/get-docker/) (to build the poppler-utils Lambda layer)
- [AWS CLI](https://aws.amazon.com/cli/) configured with credentials

## Deploy

```bash
git clone <repo-url>
cd ssooj
./deploy.sh
```

The script checks prerequisites, builds the Go binaries, extracts `pdftotext`
from an Amazon Linux 2023 Docker image, packages it as a Lambda layer, and
runs `terraform apply` to create all resources.

## Upload a Receipt

```bash
# Get the upload endpoint
ENDPOINT=$(terraform -chdir=infrastructure output -raw upload_endpoint)

# Request a presigned upload URL
UPLOAD_URL=$(curl -s -X POST "$ENDPOINT?name=receipt.pdf" | jq -r .uploadUrl)

# Upload the PDF directly to S3
curl -X PUT -H "Content-Type: application/pdf" --data-binary @receipt.pdf "$UPLOAD_URL"
```

## Check Results

```bash
# CSV output in the processed bucket
aws s3 ls s3://ssooj-receipts-processed-*/csv/ --recursive

# DynamoDB items
aws dynamodb scan --table-name ssooj-receipts
```

## Local Development

```bash
cd receipt-worker

# Test extraction against a PDF on your machine
go run ./cmd/local.go --pdf ~/Downloads/receipt.pdf
```

This runs `pdftotext -layout` on the PDF, parses the output, and dumps the
structured receipt as JSON. Useful for iterating on `receipt/parser.go`
without deploying to AWS.

## Project Layout

```
ssooj/
  presign-url/           Go Lambda -- presigned upload URL
  receipt-worker/        Go Lambda -- receipt parsing + storage
    main.go              SQS handler, PDF download, pdftotext, parse, store
    receipt/             Types, parser registry, Continente parser
    store/               CSV and DynamoDB writers
    cmd/local.go         Local test runner
  infrastructure/        Terraform definitions
    main.tf              21 AWS resources
    build-layer.sh       Builds poppler-utils Lambda layer from AL2023
  deploy.sh              One-command deploy
```

## Cost

All resources fit within the AWS Free Tier for light usage
(a few receipts per day): single t2.micro-equivalent Lambda invocations,
SQS 1M requests, 5 GB S3, 25 GB DynamoDB on-demand.
