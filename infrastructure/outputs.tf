output "raw_bucket" {
  value = aws_s3_bucket.raw.id
}

output "processed_bucket" {
  value = aws_s3_bucket.processed.id
}

output "queue_url" {
  value = aws_sqs_queue.main.url
}

output "upload_endpoint" {
  value = "${aws_apigatewayv2_api.upload.api_endpoint}/upload"
}

output "dynamodb_table" {
  value = aws_dynamodb_table.receipts.name
}

output "presign_function" {
  value = aws_lambda_function.presign.function_name
}

output "worker_function" {
  value = aws_lambda_function.worker.function_name
}

output "csv_endpoint" {
  value = "${aws_apigatewayv2_api.upload.api_endpoint}/csv"
}
