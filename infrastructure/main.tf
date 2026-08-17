locals {
  suffix = var.bucket_suffix != "" ? var.bucket_suffix : var.account_id
}

# ── S3 buckets ──────────────────────────────────────────────────────────

resource "aws_s3_bucket" "raw" {
  bucket = "ssooj-receipts-raw-${local.suffix}"
}

resource "aws_s3_bucket_cors_configuration" "raw" {
  bucket = aws_s3_bucket.raw.id
  cors_rule {
    allowed_origins = ["*"]
    allowed_methods = ["PUT", "POST"]
    allowed_headers = ["Content-Type"]
    max_age_seconds = 300
  }
}

resource "aws_s3_bucket_versioning" "raw" {
  bucket = aws_s3_bucket.raw.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket" "processed" {
  bucket = "ssooj-receipts-processed-${local.suffix}"
}

# ── SQS queues ──────────────────────────────────────────────────────────

resource "aws_sqs_queue" "dlq" {
  name = "ssooj-receipts-dlq"
}

resource "aws_sqs_queue" "main" {
  name = "ssooj-receipts-queue"
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = 3
  })
}

resource "aws_sqs_queue_policy" "s3_to_sqs" {
  queue_url = aws_sqs_queue.main.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "s3.amazonaws.com" }
      Action    = "sqs:SendMessage"
      Resource  = aws_sqs_queue.main.arn
      Condition = {
        ArnLike = {
          "aws:SourceArn" = aws_s3_bucket.raw.arn
        }
      }
    }]
  })
}

resource "aws_s3_bucket_notification" "new_pdf_to_sqs" {
  bucket = aws_s3_bucket.raw.id
  queue {
    queue_arn     = aws_sqs_queue.main.arn
    events        = ["s3:ObjectCreated:Put", "s3:ObjectCreated:CompleteMultipartUpload"]
    filter_suffix = ".pdf"
  }
  depends_on = [aws_sqs_queue_policy.s3_to_sqs]
}

# ── DynamoDB ────────────────────────────────────────────────────────────

resource "aws_dynamodb_table" "receipts" {
  name         = "ssooj-receipts"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "receipt_id"
  attribute {
    name = "receipt_id"
    type = "S"
  }
}

# ── IAM: presign URL Lambda role ────────────────────────────────────────

resource "aws_iam_role" "presign" {
  name = "ssooj-presign-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "presign" {
  name = "ssooj-presign-policy"
  role = aws_iam_role.presign.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.raw.arn}/raw/*"
      },
      {
        Effect = "Allow"
        Action = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "*"
      }
    ]
  })
}

# ── IAM: worker Lambda role ─────────────────────────────────────────────

resource "aws_iam_role" "worker" {
  name = "ssooj-worker-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "worker" {
  name = "ssooj-worker-policy"
  role = aws_iam_role.worker.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = ["s3:GetObject"]
        Resource = "${aws_s3_bucket.raw.arn}/raw/*"
      },
      {
        Effect = "Allow"
        Action = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.processed.arn}/*"
      },
      {
        Effect = "Allow"
        Action = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes"]
        Resource = aws_sqs_queue.main.arn
      },
      {
        Effect = "Allow"
        Action = ["dynamodb:PutItem"]
        Resource = aws_dynamodb_table.receipts.arn
      },
      {
        Effect = "Allow"
        Action = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "*"
      }
    ]
  })
}

# ── Lambda: presign URL ─────────────────────────────────────────────────

data "archive_file" "presign_zip" {
  type        = "zip"
  source_file = abspath("${path.module}/../presign-url/bootstrap")
  output_path = abspath("${path.module}/artifacts/presign-url.zip")
}

resource "aws_lambda_function" "presign" {
  function_name = "ssooj-presign-url"
  role          = aws_iam_role.presign.arn
  filename      = data.archive_file.presign_zip.output_path
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  timeout       = 10
  memory_size   = 128
  environment {
    variables = {
      BUCKET = aws_s3_bucket.raw.id
    }
  }
}

# ── API Gateway (HTTP API) ──────────────────────────────────────────────

resource "aws_apigatewayv2_api" "upload" {
  name          = "ssooj-upload"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "presign" {
  api_id                 = aws_apigatewayv2_api.upload.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.presign.invoke_arn
  payload_format_version = "1.0"
}

resource "aws_apigatewayv2_route" "upload_post" {
  api_id    = aws_apigatewayv2_api.upload.id
  route_key = "POST /upload"
  target    = "integrations/${aws_apigatewayv2_integration.presign.id}"
}

resource "aws_apigatewayv2_route" "upload_options" {
  api_id    = aws_apigatewayv2_api.upload.id
  route_key = "OPTIONS /upload"
  target    = "integrations/${aws_apigatewayv2_integration.presign.id}"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.upload.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_lambda_permission" "apigw_invoke_presign" {
  statement_id  = "apigw-invoke-presign"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.presign.arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.upload.execution_arn}/*/*/upload"
}

# ── Lambda Layer: poppler-utils ─────────────────────────────────────────

data "archive_file" "poppler_layer" {
  type        = "zip"
  source_dir  = abspath("${path.module}/layer/opt")
  output_path = abspath("${path.module}/artifacts/poppler-layer.zip")
}

resource "aws_lambda_layer_version" "poppler" {
  layer_name          = "poppler-utils"
  filename            = data.archive_file.poppler_layer.output_path
  compatible_runtimes = ["provided.al2023"]
}

# ── Lambda: receipt worker ──────────────────────────────────────────────

data "archive_file" "worker_zip" {
  type        = "zip"
  source_file = abspath("${path.module}/../receipt-worker/bootstrap")
  output_path = abspath("${path.module}/artifacts/receipt-worker.zip")
}

resource "aws_lambda_function" "worker" {
  function_name = "ssooj-receipt-worker"
  role          = aws_iam_role.worker.arn
  filename      = data.archive_file.worker_zip.output_path
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  timeout       = 30
  memory_size   = 256
  layers        = [aws_lambda_layer_version.poppler.arn]
  environment {
    variables = {
      RAW_BUCKET       = aws_s3_bucket.raw.id
      PROCESSED_BUCKET = aws_s3_bucket.processed.id
      DYNAMO_TABLE     = aws_dynamodb_table.receipts.name
    }
  }
  depends_on = [aws_lambda_layer_version.poppler]
}

resource "aws_lambda_event_source_mapping" "worker_sqs" {
  event_source_arn = aws_sqs_queue.main.arn
  function_name    = aws_lambda_function.worker.arn
  batch_size       = 1
}

# ── IAM: csv lister Lambda role ────────────────────────────────────────

resource "aws_iam_role" "csv" {
  name = "ssooj-csv-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "csv" {
  name = "ssooj-csv-policy"
  role = aws_iam_role.csv.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = ["s3:ListBucket"]
        Resource = aws_s3_bucket.processed.arn
      },
      {
        Effect = "Allow"
        Action = ["s3:GetObject"]
        Resource = "${aws_s3_bucket.processed.arn}/*"
      },
      {
        Effect = "Allow"
        Action = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "*"
      }
    ]
  })
}

# ── Lambda: csv lister ──────────────────────────────────────────────────

data "archive_file" "csv_zip" {
  type        = "zip"
  source_file = abspath("${path.module}/../csv-lister/bootstrap")
  output_path = abspath("${path.module}/artifacts/csv-lister.zip")
}

resource "aws_lambda_function" "csv" {
  function_name = "ssooj-csv-lister"
  role          = aws_iam_role.csv.arn
  filename      = data.archive_file.csv_zip.output_path
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  timeout       = 10
  memory_size   = 128
  environment {
    variables = {
      PROCESSED_BUCKET = aws_s3_bucket.processed.id
    }
  }
}

# ── API Gateway route: GET /csv ────────────────────────────────────────

resource "aws_apigatewayv2_integration" "csv" {
  api_id                 = aws_apigatewayv2_api.upload.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.csv.invoke_arn
  payload_format_version = "1.0"
}

resource "aws_apigatewayv2_route" "csv_get" {
  api_id    = aws_apigatewayv2_api.upload.id
  route_key = "GET /csv"
  target    = "integrations/${aws_apigatewayv2_integration.csv.id}"
}

resource "aws_lambda_permission" "apigw_invoke_csv" {
  statement_id  = "apigw-invoke-csv"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.csv.arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.upload.execution_arn}/*/*/csv"
}
