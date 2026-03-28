package config

import (
	"errors"
	"fmt"
	"os"
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
}

type Dequeue struct {
	AWS                AWS
	QueueName          string
	QueueURL           string
	DBConnectionString string
	WaitSeconds        int32
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

	return Enqueue{
		AWS:       awsCfg,
		QueueName: firstNonEmpty(getenv("QUEUE_NAME"), "sample-queue"),
		Interval:  interval,
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

	return Dequeue{
		AWS:                awsCfg,
		QueueName:          queueName,
		QueueURL:           queueURL,
		DBConnectionString: dbURL,
		WaitSeconds:        1,
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
