package message

import (
	"fmt"
	"time"

	"keda-kind/sample-app/internal/timeutil"
)

type Payload struct {
	Code   string    `json:"code"`
	SentAt time.Time `json:"sent_at"`
}

func Generate(now time.Time, randIntn func(int) int) Payload {
	return Payload{
		Code:   formatFiveDigits(10000 + randIntn(90000)),
		SentAt: timeutil.ToJST(now),
	}
}

func formatFiveDigits(value int) string {
	return fmt.Sprintf("%05d", value)
}
