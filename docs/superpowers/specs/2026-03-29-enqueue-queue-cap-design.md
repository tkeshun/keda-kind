# Enqueue Queue Capacity Check

## Goal

Prevent the sample `enqueue` app from adding an 11th message to the queue. The app may enqueue when the queue currently has 0 through 9 visible messages, and it must skip enqueueing when the queue currently has 10 or more visible messages.

## Scope

This change is limited to the sample enqueue path:

- `sample-app/internal/enqueue`
- `sample-app/internal/adapters/sqs`
- `sample-app/cmd/enqueue`
- related unit tests

No changes are required for `dequeue`, KEDA manifests, or PostgreSQL storage.

## Design

### Queue abstraction

Extend the enqueue-side `Queue` interface with a method that returns the current visible message count for a queue URL.

The count check happens after `EnsureQueue` succeeds, so the service continues to rely on the queue URL returned by the existing setup flow.

### Enqueue behavior

`Service.Tick` will:

1. Ensure the queue exists and get its URL.
2. Read the current visible message count.
3. Return without sending when the count is 10 or greater.
4. Generate and send a message only when the count is below 10.

The skip path is treated as a normal outcome, not an error.

### Result reporting

`Service.Tick` will return a small result that tells the caller whether a message was actually sent or skipped because the queue was already at capacity.

`sample-app/cmd/enqueue/main.go` will use that result to log either:

- message sent
- enqueue skipped because the queue already has 10 or more messages

### SQS adapter

The SQS adapter will fetch queue attributes and read `ApproximateNumberOfMessages`. The value will be parsed to an integer and returned to the service.

If the attribute is missing or malformed, the adapter returns an error so the service does not make a blind enqueue decision.

## Testing

Add unit tests first for:

- sending when the queue count is 9
- skipping when the queue count is 10

Existing payload assertions remain in the send case. The skip case will assert that no message body was sent.

## Error Handling

- queue creation failures still fail the tick
- queue count lookup failures still fail the tick
- JSON marshal failures still fail the tick
- skip because count is already 10 or more is not an error

## Non-Goals

- exact distributed queue limit enforcement across multiple concurrent enqueue workers
- changes to KEDA scaling thresholds
- changes to dequeue throughput
