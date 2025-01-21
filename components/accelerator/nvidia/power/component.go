// Package power tracks the NVIDIA per-GPU power usage.
package power

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/leptonai/gpud/components"
	nvidia_common "github.com/leptonai/gpud/components/accelerator/nvidia/common"
	nvidia_power_id "github.com/leptonai/gpud/components/accelerator/nvidia/power/id"
	nvidia_query "github.com/leptonai/gpud/components/accelerator/nvidia/query"
	nvidia_query_metrics_power "github.com/leptonai/gpud/components/accelerator/nvidia/query/metrics/power"
	"github.com/leptonai/gpud/components/query"
	"github.com/leptonai/gpud/log"

	"github.com/prometheus/client_golang/prometheus"
)

func New(ctx context.Context, cfg nvidia_common.Config) components.Component {
	cfg.Query.SetDefaultsIfNotSet()

	cctx, ccancel := context.WithCancel(ctx)
	nvidia_query.SetDefaultPoller(
		nvidia_query.WithDBRW(cfg.Query.State.DBRW),
		nvidia_query.WithDBRO(cfg.Query.State.DBRO),
		nvidia_query.WithNvidiaSMICommand(cfg.NvidiaSMICommand),
		nvidia_query.WithNvidiaSMIQueryCommand(cfg.NvidiaSMIQueryCommand),
		nvidia_query.WithIbstatCommand(cfg.IbstatCommand),
		nvidia_query.WithInfinibandClassDirectory(cfg.InfinibandClassDirectory),
	)
	nvidia_query.GetDefaultPoller().Start(cctx, cfg.Query, nvidia_power_id.Name)

	return &component{
		rootCtx: ctx,
		cancel:  ccancel,
		poller:  nvidia_query.GetDefaultPoller(),
	}
}

var _ components.Component = (*component)(nil)

type component struct {
	rootCtx  context.Context
	cancel   context.CancelFunc
	poller   query.Poller
	gatherer prometheus.Gatherer
}

func (c *component) Name() string { return nvidia_power_id.Name }

func (c *component) States(ctx context.Context) ([]components.State, error) {
	last, err := c.poller.LastSuccess()
	if err == query.ErrNoData { // no data
		log.Logger.Debugw("nothing found in last state (no data collected yet)", "component", nvidia_power_id.Name)
		return []components.State{
			{
				Name:    nvidia_power_id.Name,
				Healthy: true,
				Reason:  query.ErrNoData.Error(),
			},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	allOutput, ok := last.Output.(*nvidia_query.Output)
	if !ok {
		return nil, fmt.Errorf("invalid output type: %T", last.Output)
	}
	if lerr := c.poller.LastError(); lerr != nil {
		log.Logger.Warnw("last query failed -- returning cached, possibly stale data", "error", lerr)
	}
	lastSuccessPollElapsed := time.Now().UTC().Sub(allOutput.Time)
	if lastSuccessPollElapsed > 2*c.poller.Config().Interval.Duration {
		log.Logger.Warnw("last poll is too old", "elapsed", lastSuccessPollElapsed, "interval", c.poller.Config().Interval.Duration)
	}

	if allOutput.SMIExists && len(allOutput.SMIQueryErrors) > 0 {
		cs := make([]components.State, 0)
		for _, e := range allOutput.SMIQueryErrors {
			cs = append(cs, components.State{
				Name:    nvidia_power_id.Name,
				Healthy: false,
				Error:   e,
				Reason:  "nvidia-smi query failed with " + e,
				ExtraInfo: map[string]string{
					nvidia_query.StateKeySMIExists: fmt.Sprintf("%v", allOutput.SMIExists),
				},
			})
		}
		return cs, nil
	}
	output := ToOutput(allOutput)
	return output.States()
}

func (c *component) Events(ctx context.Context, since time.Time) ([]components.Event, error) {
	return nil, nil
}

func (c *component) Metrics(ctx context.Context, since time.Time) ([]components.Metric, error) {
	log.Logger.Debugw("querying metrics", "since", since)

	currentUsageMilliWatts, err := nvidia_query_metrics_power.ReadCurrentUsageMilliWatts(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to read current usage milli watts: %w", err)
	}
	enforcedLimitMilliWatts, err := nvidia_query_metrics_power.ReadEnforcedLimitMilliWatts(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to read enforced limit milli watts: %w", err)
	}
	usedPercents, err := nvidia_query_metrics_power.ReadUsedPercents(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to read used percents: %w", err)
	}

	ms := make([]components.Metric, 0, len(currentUsageMilliWatts)+len(enforcedLimitMilliWatts)+len(usedPercents))
	for _, m := range currentUsageMilliWatts {
		ms = append(ms, components.Metric{
			Metric: m,
			ExtraInfo: map[string]string{
				"gpu_id": m.MetricSecondaryName,
			},
		})
	}
	for _, m := range enforcedLimitMilliWatts {
		ms = append(ms, components.Metric{
			Metric: m,
			ExtraInfo: map[string]string{
				"gpu_id": m.MetricSecondaryName,
			},
		})
	}
	for _, m := range usedPercents {
		ms = append(ms, components.Metric{
			Metric: m,
			ExtraInfo: map[string]string{
				"gpu_id": m.MetricSecondaryName,
			},
		})
	}

	return ms, nil
}

func (c *component) Close() error {
	log.Logger.Debugw("closing component")

	// safe to call stop multiple times
	_ = c.poller.Stop(nvidia_power_id.Name)

	return nil
}

var _ components.PromRegisterer = (*component)(nil)

func (c *component) RegisterCollectors(reg *prometheus.Registry, dbRW *sql.DB, dbRO *sql.DB, tableName string) error {
	c.gatherer = reg
	return nvidia_query_metrics_power.Register(reg, dbRW, dbRO, tableName)
}
