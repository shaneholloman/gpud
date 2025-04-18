package state

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"math/rand"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	components "github.com/leptonai/gpud/api/v1"
	"github.com/leptonai/gpud/pkg/sqlite"
)

func TestState(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tableName := "test_metrics"
	if err := CreateTableMetrics(ctx, db, tableName); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	if readMetric, err := ReadLastMetric(ctx, db, tableName, "test_metric", ""); readMetric != nil || err != nil {
		t.Errorf("expected no metric + no error, got %v, %v", readMetric, err)
	}

	// test AvgSince with no rows
	since := time.Now().Add(-1 * time.Hour)
	avg, err := AvgSince(ctx, db, tableName, "test_metric", "", since)
	if err != nil {
		t.Errorf("AvgSince failed with no rows: %v", err)
	}
	if avg != 0 {
		t.Errorf("expected average 0 with no rows, got %f", avg)
	}

	now := time.Now()
	metrics := []components.Metric{
		{UnixSeconds: now.Unix(), DeprecatedMetricName: "test_metric", Value: 10.0},
		{UnixSeconds: now.Add(-1 * time.Minute).Unix(), DeprecatedMetricName: "test_metric", Value: 20.0},
		{UnixSeconds: now.Add(-2 * time.Minute).Unix(), DeprecatedMetricName: "test_metric", Value: 30.0},
	}

	for _, m := range metrics {
		if err := InsertMetric(ctx, db, tableName, m); err != nil {
			t.Errorf("failed to insert metric: %v", err)
			return
		}
	}

	readMetric, err := ReadLastMetric(ctx, db, tableName, "test_metric", "")
	if err != nil {
		t.Errorf("failed to read last metric: %v", err)
		return
	}

	if readMetric.Value != metrics[0].Value {
		t.Errorf("expected value %f, got %f", metrics[0].Value, readMetric.Value)
	}
	if readMetric.UnixSeconds != metrics[0].UnixSeconds {
		t.Errorf("expected time %v, got %v", metrics[0].UnixSeconds, readMetric.UnixSeconds)
	}

	since = time.Now().Add(-3 * time.Minute)
	avg, err = AvgSince(ctx, db, tableName, "test_metric", "", since)
	if err != nil {
		t.Errorf("failed to get average: %v", err)
	}

	expectedAvg := (10.0 + 20.0 + 30.0) / 3.0
	if math.Abs(avg-expectedAvg) > 0.001 {
		t.Errorf("expected average %f, got %f", expectedAvg, avg)
	}
	t.Logf("expected avg: %f, avg: %f", expectedAvg, avg)
}

func TestStateMoreDataPoints(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tableName := "test_metrics"

	if err := CreateTableMetrics(ctx, db, tableName); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	if readMetric, err := ReadLastMetric(ctx, db, tableName, "test_metric", ""); readMetric != nil || err != nil {
		t.Errorf("expected no metric + no error, got %v, %v", readMetric, err)
	}

	// test AvgSince with no rows
	since := time.Now().Add(-1 * time.Hour)
	avg, err := AvgSince(ctx, db, tableName, "test_metric", "", since)
	if err != nil {
		t.Errorf("AvgSince failed with no rows: %v", err)
	}
	if avg != 0 {
		t.Errorf("expected average 0 with no rows, got %f", avg)
	}

	now := time.Now()
	metricNames := []string{"cpu_usage", "memory_usage", "disk_io", "network_traffic"}
	numDataPoints := 1000

	for i := numDataPoints - 1; i >= 0; i-- {
		metricName := metricNames[rand.Intn(len(metricNames))]
		timestamp := metav1.NewTime(now.Add(time.Duration(-i) * time.Minute))
		value := rand.Float64() * 100

		metric := components.Metric{
			UnixSeconds:                   timestamp.Unix(),
			DeprecatedMetricName:          metricName,
			DeprecatedMetricSecondaryName: "",
			Value:                         value,
		}
		if err := InsertMetric(ctx, db, tableName, metric); err != nil {
			t.Fatalf("failed to insert metric: %v", err)
		}
	}

	for _, metricName := range metricNames {
		since := now.Add(-30 * time.Minute)
		readMetrics, err := ReadMetricsSince(ctx, db, tableName, metricName, "", since)
		if err != nil {
			t.Errorf("failed to read metrics for %s since %v: %v", metricName, since, err)
			continue
		}

		if len(readMetrics) == 0 {
			t.Errorf("expected some metrics for %s, got none", metricName)
			continue
		}

		for i := 1; i < len(readMetrics); i++ {
			if readMetrics[i].UnixSeconds < readMetrics[i-1].UnixSeconds {
				t.Errorf("Metrics for %s are not in chronological order", metricName)
			}
			if readMetrics[i].UnixSeconds < since.Unix() {
				t.Errorf("Metric for %s is before the 'since' time", metricName)
			}
		}

		t.Logf("Read %d metrics for %s", len(readMetrics), metricName)

		avg, err := AvgSince(ctx, db, tableName, metricName, "", since)
		if err != nil {
			t.Errorf("failed to get average for %s since %v: %v", metricName, since, err)
		} else {
			var sum float64
			for _, m := range readMetrics {
				sum += m.Value
			}
			expectedAvg := sum / float64(len(readMetrics))
			if math.Abs(avg-expectedAvg) > 0.001 {
				t.Errorf("Average for %s doesn't match: expected %.3f, got %.3f", metricName, expectedAvg, avg)
			} else {
				t.Logf("Average for %s: %.3f", metricName, avg)
			}
		}
	}

	purgeTime := now.Add(-200 * time.Minute)
	purged, err := PurgeMetrics(ctx, db, tableName, purgeTime)
	if err != nil {
		t.Fatalf("failed to purge data: %v", err)
	}
	t.Logf("purged %d metrics", purged)

	if purged != 799 {
		t.Errorf("expected 799 metrics to be purged, got %d", purged)
	}

	purgeTime = now.Add(5 * time.Minute)
	purged, err = PurgeMetrics(ctx, db, tableName, purgeTime)
	if err != nil {
		t.Fatalf("failed to purge data: %v", err)
	}
	t.Logf("purged %d metrics", purged)
	if purged != 201 {
		t.Errorf("expected 201 metrics to be purged, got %d", purged)
	}

	for _, metricName := range metricNames {
		readMetrics, err := ReadMetricsSince(ctx, db, tableName, metricName, "", purgeTime)
		if errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected no data for %s, got %d", metricName, len(readMetrics))
		}
	}

	for _, metricName := range metricNames {
		avg, err := AvgSince(ctx, db, tableName, metricName, "", purgeTime)
		if err != nil {
			t.Errorf("failed to get average for %s after purge: %v", metricName, err)
		} else if avg != 0.0 {
			t.Errorf("expected average 0.0 for %s after purge, got %.3f", metricName, avg)
		}
	}

	secondaryID := "test_secondary_id"
	secondaryMetrics := []components.Metric{
		{UnixSeconds: now.Add(-5 * time.Minute).Unix(), DeprecatedMetricName: "test_metric", DeprecatedMetricSecondaryName: secondaryID, Value: 10.0},
		{UnixSeconds: now.Add(-4 * time.Minute).Unix(), DeprecatedMetricName: "test_metric", DeprecatedMetricSecondaryName: secondaryID, Value: 20.0},
		{UnixSeconds: now.Add(-3 * time.Minute).Unix(), DeprecatedMetricName: "test_metric", DeprecatedMetricSecondaryName: secondaryID, Value: 30.0},
		{UnixSeconds: now.Add(-2 * time.Minute).Unix(), DeprecatedMetricName: "test_metric", DeprecatedMetricSecondaryName: secondaryID, Value: 40.0},
		{UnixSeconds: now.Add(-1 * time.Minute).Unix(), DeprecatedMetricName: "test_metric", DeprecatedMetricSecondaryName: secondaryID, Value: 50.0},
	}

	for _, m := range secondaryMetrics {
		if err := InsertMetric(ctx, db, tableName, m); err != nil {
			t.Fatalf("failed to insert metric with secondary ID: %v", err)
		}
	}

	// Test AvgSince with secondary ID
	secondarySince := now.Add(-6 * time.Minute)
	secondaryAvg, err := AvgSince(ctx, db, tableName, "test_metric", secondaryID, secondarySince)
	if err != nil {
		t.Errorf("AvgSince failed with secondary ID: %v", err)
	} else {
		expectedAvg := 30.0 // (10 + 20 + 30 + 40 + 50) / 5
		if math.Abs(secondaryAvg-expectedAvg) > 0.001 {
			t.Errorf("Average with secondary ID doesn't match: expected %.3f, got %.3f", expectedAvg, secondaryAvg)
		} else {
			t.Logf("Average with secondary ID: %.3f", secondaryAvg)
		}
	}
}

func TestAvgSincePanicWithEmptySecondaryName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tableName := "test_metrics_panic"
	if err := CreateTableMetrics(ctx, db, tableName); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	avg, err := AvgSince(ctx, db, tableName, "test_panic_metric", "", time.Now().Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("failed to get average: %v", err)
	}
	if avg != 0.0 {
		t.Errorf("expected average 0.0, got %.3f", avg)
	}

	// Insert a metric with an empty secondary name
	now := time.Now()
	metric := components.Metric{
		UnixSeconds:                   now.Unix(),
		DeprecatedMetricName:          "test_panic_metric",
		DeprecatedMetricSecondaryName: "", // Empty secondary name
		Value:                         42.0,
	}

	if err := InsertMetric(ctx, db, tableName, metric); err != nil {
		t.Fatalf("failed to insert metric: %v", err)
	}

	avg, err = AvgSince(ctx, db, tableName, "test_panic_metric", "", time.Now().Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("failed to get average: %v", err)
	}
	if avg != 42.0 {
		t.Errorf("expected average 42.0, got %.3f", avg)
	}
}
