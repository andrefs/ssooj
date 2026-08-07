AWS_REGION ?= eu-west-1
FUNCTION_NAME ?= ssooj-receipt-worker

.PHONY: build deploy local

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
	zip function.zip bootstrap

deploy: build
	aws lambda update-function-code \
		--function-name $(FUNCTION_NAME) \
		--zip-file fileb://function.zip \
		--region $(AWS_REGION)

local:
	go run ./cmd/local.go --pdf $(PDF)
