package sqs

import (
	"context"
	"fmt"
	"strconv"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"keda-kind/sample-app/internal/config"
	"keda-kind/sample-app/internal/dequeue"
)

type Client struct {
	api *sqs.Client
}

func New(ctx context.Context, cfg config.AWS) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	api := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		o.BaseEndpoint = &cfg.Endpoint
	})

	return &Client{api: api}, nil
}

func (c *Client) EnsureQueue(ctx context.Context, queueName string) (string, error) {
	out, err := c.api.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: &queueName,
	})
	if err != nil {
		return "", fmt.Errorf("create queue: %w", err)
	}
	return *out.QueueUrl, nil
}

func (c *Client) SendMessage(ctx context.Context, queueURL string, body string) error {
	_, err := c.api.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &queueURL,
		MessageBody: &body,
	})
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

func (c *Client) VisibleMessageCount(ctx context.Context, queueURL string) (int, error) {
	// SQS exposes an approximate visible message count. This is sufficient for the
	// sample app's best-effort enqueue throttling, but it is not a strict cap under churn.
	out, err := c.api.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: &queueURL,
		AttributeNames: []sqstypes.QueueAttributeName{
			sqstypes.QueueAttributeNameApproximateNumberOfMessages,
		},
	})
	if err != nil {
		return 0, fmt.Errorf("get queue attributes: %w", err)
	}

	rawCount, ok := out.Attributes[string(sqstypes.QueueAttributeNameApproximateNumberOfMessages)]
	if !ok || rawCount == "" {
		return 0, fmt.Errorf("missing approximate number of messages for queue %q", queueURL)
	}

	count, err := strconv.Atoi(rawCount)
	if err != nil {
		return 0, fmt.Errorf("parse approximate number of messages %q: %w", rawCount, err)
	}

	return count, nil
}

func (c *Client) ReceiveOne(ctx context.Context, queueURL string, waitSeconds int32) (*dequeue.QueueMessage, error) {
	out, err := c.api.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     waitSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("receive message: %w", err)
	}
	if len(out.Messages) == 0 {
		return nil, nil
	}

	message := out.Messages[0]
	return &dequeue.QueueMessage{
		Body:          stringValue(message.Body),
		ReceiptHandle: stringValue(message.ReceiptHandle),
	}, nil
}

func (c *Client) Delete(ctx context.Context, queueURL string, receiptHandle string) error {
	_, err := c.api.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: &receiptHandle,
	})
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	return nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
