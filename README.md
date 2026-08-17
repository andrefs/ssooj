[![CI](https://github.com/andrefs/ssooj/actions/workflows/ci.yml/badge.svg)](https://github.com/andrefs/ssooj/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25-blue)](https://go.dev/)

# ssooj -- Supermarket Receipt PDF Parser

> **Disclaimer:** This is a wildly overengineered solution for extracting
> data from supermarket receipts. Instead of reading the PDF with your
> eyes and typing the numbers into a spreadsheet, we built an
> event-driven serverless pipeline spanning S3, SQS, Lambda, DynamoDB,
> API Gateway, and Terraform. But it works.

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

## Web Upload Form

`upload-form/` contains a static HTML page for uploading receipts from a
browser (drag-and-drop, multi-file). It is generated from a template so the
API Gateway ID is baked in at build time instead of being committed.

```bash
cd upload-form

# 1. Configure the API ID (not committed, see .env.example)
cp .env.example .env
# edit .env and set API_ID=<your api id>

# 2. Generate the page
go run ./gen   # writes dist/index.html
```

The generator fails if `API_ID` is missing or still the placeholder.

### Deploying to Komodo

The `upload-form/Dockerfile` is a multi-stage build that runs the generator
inside the build and serves the page with nginx. Build it with the `.env`
passed as a BuildKit secret so the API ID never lands in an image layer or
in git:

```bash
cd upload-form
docker build --secret id=ssooj_env,src=.env -t ssooj-upload .
```

On Komodo, point a service at the `upload-form/` directory and configure the
build secret `ssooj_env` with your `.env` contents.

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
  upload-form/           Static upload page (browser)
    index.tmpl.html      Page template (API ID placeholder)
    gen/                 Go generator: reads .env, renders dist/index.html
    Dockerfile           nginx image; runs generator at build time
  deploy.sh              One-command deploy
```

## Cost

All resources fit within the AWS Free Tier for light usage
(a few receipts per day): single t2.micro-equivalent Lambda invocations,
SQS 1M requests, 5 GB S3, 25 GB DynamoDB on-demand.
