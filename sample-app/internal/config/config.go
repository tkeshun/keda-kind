package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type AWS struct {
	Endpoint string
	Region   string
}

type Enqueue struct {
	AWS       AWS
	QueueName string
	Interval  time.Duration
	Mode      string
	HTTPPort  int
}

type Dequeue struct {
	AWS                AWS
	QueueName          string
	QueueURL           string
	DBConnectionString string
	WaitSeconds        int32
	StoreDelay         time.Duration
}

func LoadEnqueue(getenv func(string) string) (Enqueue, error) {
	awsCfg, err := loadAWS(getenv)
	if err != nil {
		return Enqueue{}, err
	}

	interval := 5 * time.Second
	if raw := getenv("SEND_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return Enqueue{}, fmt.Errorf("parse SEND_INTERVAL: %w", err)
		}
		interval = parsed
	}

	mode := firstNonEmpty(getenv("ENQUEUE_MODE"), "scheduled")
	switch mode {
	case "scheduled", "http":
	default:
		return Enqueue{}, fmt.Errorf("invalid ENQUEUE_MODE: %s", mode)
	}

	httpPort := 8080
	if raw := getenv("HTTP_PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Enqueue{}, fmt.Errorf("parse HTTP_PORT: %w", err)
		}
		httpPort = parsed
	}

	return Enqueue{
		AWS:       awsCfg,
		QueueName: firstNonEmpty(getenv("QUEUE_NAME"), "sample-queue"),
		Interval:  interval,
		Mode:      mode,
		HTTPPort:  httpPort,
	}, nil
}

func LoadDequeue(getenv func(string) string) (Dequeue, error) {
	awsCfg, err := loadAWS(getenv)
	if err != nil {
		return Dequeue{}, err
	}

	dbURL := getenv("DB_CONNECTION_STRING")
	if dbURL == "" {
		return Dequeue{}, errors.New("DB_CONNECTION_STRING is required")
	}

	queueName := firstNonEmpty(getenv("QUEUE_NAME"), "sample-queue")
	queueURL := getenv("QUEUE_URL")
	if queueURL == "" {
		queueURL = fmt.Sprintf("%s/queue/%s", trimTrailingSlash(awsCfg.Endpoint), queueName)
	}

	storeDelay := time.Duration(0)
	if raw := getenv("DEQUEUE_STORE_DELAY_SECONDS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Dequeue{}, fmt.Errorf("parse DEQUEUE_STORE_DELAY_SECONDS: %w", err)
		}
		if parsed < 0 {
			return Dequeue{}, errors.New("DEQUEUE_STORE_DELAY_SECONDS must be >= 0")
		}
		storeDelay = time.Duration(parsed) * time.Second
	}

	return Dequeue{
		AWS:                awsCfg,
		QueueName:          queueName,
		QueueURL:           queueURL,
		DBConnectionString: dbURL,
		WaitSeconds:        1,
		StoreDelay:         storeDelay,
	}, nil
}

func loadAWS(getenv func(string) string) (AWS, error) {
	endpoint := getenv("AWS_ENDPOINT")
	if endpoint == "" {
		return AWS{}, errors.New("AWS_ENDPOINT is required")
	}

	return AWS{
		Endpoint: endpoint,
		Region:   firstNonEmpty(getenv("AWS_REGION"), "elasticmq"),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func trimTrailingSlash(value string) string {
	for len(value) > 0 && value[len(value)-1] == '/' {
		value = value[:len(value)-1]
	}
	return value
}

func MustLoadEnqueue() Enqueue {
	cfg, err := LoadEnqueue(os.Getenv)
	if err != nil {
		panic(err)
	}
	return cfg
}

func MustLoadDequeue() Dequeue {
	cfg, err := LoadDequeue(os.Getenv)
	if err != nil {
		panic(err)
	}
	return cfg
}
