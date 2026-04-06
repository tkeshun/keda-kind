package message

import (
	"encoding/json"
	"testing"
	"time"

	"keda-kind/sample-app/internal/timeutil"
)

func TestGenerateProducesFiveDigitCodeAndTimestamp(t *testing.T) {
	now := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)

	got := Generate(now, func(max int) int {
		if max != 90000 {
			t.Fatalf("unexpected max: %d", max)
		}
		return 42
	})

	if got.Code != "10042" {
		t.Fatalf("unexpected code: %s", got.Code)
	}
	expected := timeutil.ToJST(now)
	if !got.SentAt.Equal(expected) {
		t.Fatalf("unexpected timestamp: %s", got.SentAt)
	}
}

func TestPayloadJSONRoundTrip(t *testing.T) {
	payload := Payload{
		Code:   "54321",
		SentAt: time.Date(2026, 3, 29, 10, 5, 0, 0, time.UTC),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got Payload
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got != payload {
		t.Fatalf("unexpected payload: %#v", got)
	}
}
