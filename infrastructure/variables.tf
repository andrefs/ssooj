variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "eu-west-1"
}

variable "account_id" {
  description = "AWS account ID"
  type        = string
  nullable    = false
}

variable "bucket_suffix" {
  description = "Suffix for S3 bucket names (e.g. account ID)"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Common tags"
  type        = map(string)
  default = {
    Project     = "ssooj"
    ManagedBy   = "terraform"
  }
}
