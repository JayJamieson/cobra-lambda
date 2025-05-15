terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.80.0"
    }
  }
}

data "aws_caller_identity" "current" {}

provider "aws" {
  region = var.region
}

data "archive_file" "node_lambda" {
  type        = "zip"
  source_dir = "./node-demo"
  output_path = "function.zip"
}

data "archive_file" "go_lambda" {
  type        = "zip"
  source_file = "./go-demo/bootstrap"
  output_path = "bootstrap.zip"
}

resource "aws_lambda_function" "cobra_node_lambda" {
  function_name = "cobra-node-demo"

  count = 0

  role             = aws_iam_role.lambda_iam_role.arn
  handler          = "index.handler"
  architectures    = ["x86_64"]
  filename         = "function.zip"
  source_code_hash = data.archive_file.node_lambda.output_base64sha256
  runtime          = "nodejs20.x"
  timeout          = 900
  depends_on = [
    aws_iam_role_policy_attachment.lambda_policy_attachment
  ]

}

resource "aws_lambda_function" "cobra_go_lambda" {
  function_name = "cobra-go-demo"

  role             = aws_iam_role.lambda_iam_role.arn
  handler          = "bootstrap"
  architectures    = ["x86_64"]
  filename         = "bootstrap.zip"
  source_code_hash = data.archive_file.go_lambda.output_base64sha256
  runtime          = "provided.al2"
  timeout          = 60
  depends_on = [
    aws_iam_role_policy_attachment.lambda_policy_attachment
  ]
}

resource "aws_iam_role" "lambda_iam_role" {
  name               = "lambda-execution-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role_policy.json
}

data "aws_iam_policy_document" "lambda_assume_role_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_policy" "lambda_execution_policy" {
  name        = "demo-lambda-basic-execution"
  description = "policy to allow basic execution of lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = ["logs:CreateLogGroup"]
        Effect = "Allow"
        Resource = [
          "arn:aws:logs:${var.region}:${data.aws_caller_identity.current.account_id}:*"
        ]
      },
      {
        Effect = "Allow",
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ],
        Resource = "arn:aws:logs:${var.region}:${data.aws_caller_identity.current.account_id}:log-group:*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_policy_attachment" {
  role       = aws_iam_role.lambda_iam_role.name
  policy_arn = aws_iam_policy.lambda_execution_policy.arn
}

output "lambda_function_arn" {
  value = aws_lambda_function.cobra_go_lambda.invoke_arn
}

output "lambda_function_name" {
  value = aws_lambda_function.cobra_go_lambda.function_name
}
