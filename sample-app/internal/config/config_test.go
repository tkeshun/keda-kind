package config

import (
	"testing"
	"time"
)

func TestLoadEnqueueUsesDefaultsAndRequiredEndpoint(t *testing.T) {
	cfg, err := LoadEnqueue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.QueueName != "sample-queue" {
		t.Fatalf("unexpected queue name: %s", cfg.QueueName)
	}
	if cfg.AWS.Region != "elasticmq" {
		t.Fatalf("unexpected region: %s", cfg.AWS.Region)
	}
	if cfg.Interval != 5*time.Second {
		t.Fatalf("unexpected interval: %s", cfg.Interval)
	}
}

func TestLoadEnqueueRequiresEndpoint(t *testing.T) {
	_, err := LoadEnqueue(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected an error when AWS_ENDPOINT is missing")
	}
}

func TestLoadDequeueUsesDefaultsAndRequiresDatabaseURL(t *testing.T) {
	cfg, err := LoadDequeue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		case "DB_CONNECTION_STRING":
			return "postgres://example"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.QueueName != "sample-queue" {
		t.Fatalf("unexpected queue name: %s", cfg.QueueName)
	}
	if cfg.AWS.Region != "elasticmq" {
		t.Fatalf("unexpected region: %s", cfg.AWS.Region)
	}
	if cfg.QueueURL != "http://elasticmq:9324/queue/sample-queue" {
		t.Fatalf("unexpected queue url: %s", cfg.QueueURL)
	}
	if cfg.WaitSeconds != 1 {
		t.Fatalf("unexpected wait seconds: %d", cfg.WaitSeconds)
	}
}

func TestLoadDequeueRequiresDatabaseURL(t *testing.T) {
	_, err := LoadDequeue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		default:
			return ""
		}
	})
	if err == nil {
		t.Fatal("expected an error when DB_CONNECTION_STRING is missing")
	}
}
