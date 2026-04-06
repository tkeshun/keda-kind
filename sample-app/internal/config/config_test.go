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
	if cfg.Mode != "scheduled" {
		t.Fatalf("unexpected mode: %s", cfg.Mode)
	}
	if cfg.HTTPPort != 8080 {
		t.Fatalf("unexpected http port: %d", cfg.HTTPPort)
	}
}

func TestLoadEnqueueRequiresEndpoint(t *testing.T) {
	_, err := LoadEnqueue(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected an error when AWS_ENDPOINT is missing")
	}
}

func TestLoadEnqueueReadsHTTPModeAndPort(t *testing.T) {
	cfg, err := LoadEnqueue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		case "ENQUEUE_MODE":
			return "http"
		case "HTTP_PORT":
			return "9090"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Mode != "http" {
		t.Fatalf("unexpected mode: %s", cfg.Mode)
	}
	if cfg.HTTPPort != 9090 {
		t.Fatalf("unexpected http port: %d", cfg.HTTPPort)
	}
}

func TestLoadEnqueueRejectsInvalidMode(t *testing.T) {
	_, err := LoadEnqueue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		case "ENQUEUE_MODE":
			return "invalid"
		default:
			return ""
		}
	})
	if err == nil {
		t.Fatal("expected an error when ENQUEUE_MODE is invalid")
	}
}

func TestLoadEnqueueRejectsInvalidHTTPPort(t *testing.T) {
	_, err := LoadEnqueue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		case "HTTP_PORT":
			return "not-a-number"
		default:
			return ""
		}
	})
	if err == nil {
		t.Fatal("expected an error when HTTP_PORT is invalid")
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
	if cfg.StoreDelay != 0 {
		t.Fatalf("unexpected store delay: %s", cfg.StoreDelay)
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

func TestLoadDequeueReadsStoreDelaySeconds(t *testing.T) {
	cfg, err := LoadDequeue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		case "DB_CONNECTION_STRING":
			return "postgres://example"
		case "DEQUEUE_STORE_DELAY_SECONDS":
			return "5"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.StoreDelay != 5*time.Second {
		t.Fatalf("unexpected store delay: %s", cfg.StoreDelay)
	}
}

func TestLoadDequeueRejectsInvalidStoreDelaySeconds(t *testing.T) {
	_, err := LoadDequeue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		case "DB_CONNECTION_STRING":
			return "postgres://example"
		case "DEQUEUE_STORE_DELAY_SECONDS":
			return "bad"
		default:
			return ""
		}
	})
	if err == nil {
		t.Fatal("expected an error when DEQUEUE_STORE_DELAY_SECONDS is invalid")
	}
}
