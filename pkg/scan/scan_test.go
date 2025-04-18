package scan

import (
	"context"
	"testing"
	"time"
)

func TestScan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := Scan(ctx, WithNetcheck(false), WithKMsgCheck(true)); err != nil {
		t.Logf("error scanning: %+v", err)
	}
}
