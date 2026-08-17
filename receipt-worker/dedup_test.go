package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func newTestDDb(t *testing.T, handler http.HandlerFunc) *dynamodb.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return dynamodb.NewFromConfig(aws.Config{
		Region: "eu-west-1",
	}, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
	})
}

func TestClaimHashSuccess(t *testing.T) {
	var calls int
	client := newTestDDb(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls > 1 {
			t.Fatalf("expected only one PutItem, got %d", calls)
		}
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	claimed, err := claimHash(context.Background(), client, "ssooj-receipt-hashes", "abc123")
	if err != nil {
		t.Fatalf("claimHash: %v", err)
	}
	if !claimed {
		t.Fatal("expected claim to succeed")
	}
}

func TestClaimHashDuplicate(t *testing.T) {
	client := newTestDDb(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		w.Header().Set("X-Amzn-Errortype", "com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"The conditional request failed"}`))
	})

	claimed, err := claimHash(context.Background(), client, "ssooj-receipt-hashes", "abc123")
	if err != nil {
		t.Fatalf("claimHash: %v", err)
	}
	if claimed {
		t.Fatal("expected claim to fail (duplicate)")
	}
}

func TestClaimHashServerError(t *testing.T) {
	client := newTestDDb(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"__type":"com.amazonaws.dynamodb.v20120810#InternalServerError","message":"boom"}`))
	})

	if _, err := claimHash(context.Background(), client, "ssooj-receipt-hashes", "abc123"); err == nil {
		t.Fatal("expected error for server failure")
	}
}

func TestReleaseHash(t *testing.T) {
	var body string
	client := newTestDDb(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	if err := releaseHash(context.Background(), client, "ssooj-receipt-hashes", "abc123"); err != nil {
		t.Fatalf("releaseHash: %v", err)
	}
	if !strings.Contains(body, "abc123") {
		t.Fatalf("expected hash in delete request, got body: %s", body)
	}
}
