AWS_REGION ?= eu-west-1
FUNCTION_NAME ?= ssooj-receipt-worker
LAYER_ARN ?= arn:aws:lambda:eu-west-1:ACCOUNT_ID:layer:poppler-utils:1

.PHONY: build deploy local

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
	zip function.zip bootstrap

deploy: build
	aws lambda update-function-code \
		--function-name $(FUNCTION_NAME) \
		--zip-file fileb://function.zip \
		--region $(AWS_REGION)

deploy-layer:
	cd /tmp/layer-out && zip -r /tmp/poppler-layer.zip opt/ && \
	aws lambda publish-layer-version \
		--layer-name poppler-utils \
		--zip-file fileb:///tmp/poppler-layer.zip \
		--compatible-runtimes provided.al2023 \
		--region $(AWS_REGION)

local:
	go run ./cmd/local.go --pdf $(PDF)
