package tail

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"github.com/leptonai/gpud/pkg/log"
	"github.com/leptonai/gpud/pkg/process"
	query_log_common "github.com/leptonai/gpud/pkg/query/log/common"

	"github.com/nxadm/tail"
)

func NewFromCommand(ctx context.Context, commands [][]string, opts ...OpOption) (Streamer, error) {
	op := &Op{
		commands: commands,
	}
	if err := op.ApplyOpts(opts); err != nil {
		return nil, err
	}

	processOpts := []process.OpOption{process.WithCommands(op.commands), process.WithRunAsBashScript()}
	for k, v := range op.labels {
		processOpts = append(processOpts, process.WithLabel(k, v))
	}
	p, err := process.New(processOpts...)
	if err != nil {
		return nil, err
	}
	if err := p.Start(ctx); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-p.Wait():
		return nil, fmt.Errorf("command exited unexpectedly: %w", err)
	case <-time.After(50 * time.Millisecond):
	}

	stdoutScanner := bufio.NewScanner(p.StdoutReader())
	stderrScanner := bufio.NewScanner(p.StderrReader())

	streamer := &commandStreamer{
		op:            op,
		ctx:           ctx,
		proc:          p,
		lineC:         make(chan Line, 200),
		dedupEnabled:  op.dedup,
		skipEmptyLine: op.skipEmptyLine,
	}

	if op.dedup {
		streamer.dedup = seenPool.Get().(*streamDeduper)
	}

	go streamer.pollLoops(stdoutScanner)
	go streamer.pollLoops(stderrScanner)
	go streamer.waitCommand()

	return streamer, nil
}

var _ Streamer = (*commandStreamer)(nil)

type commandStreamer struct {
	op    *Op
	ctx   context.Context
	proc  process.Process
	lineC chan Line

	dedupEnabled  bool
	dedup         *streamDeduper
	skipEmptyLine bool
}

func (sr *commandStreamer) File() string {
	return ""
}

func (sr *commandStreamer) Commands() [][]string {
	return sr.op.commands
}

func (sr *commandStreamer) Line() <-chan Line {
	return sr.lineC
}

func (sr *commandStreamer) pollLoops(scanner *bufio.Scanner) {
	var (
		err           error
		shouldInclude bool
		matchedFilter *query_log_common.Filter
	)

	for scanner.Scan() {
		select {
		case <-sr.ctx.Done():
			return
		default:
		}

		txt := scanner.Text()

		if len(txt) == 0 && sr.skipEmptyLine {
			continue
		}

		if sr.dedupEnabled {
			sr.dedup.mu.Lock()
			_, exists := sr.dedup.seen[txt]
			if exists {
				sr.dedup.mu.Unlock()
				continue
			}
			sr.dedup.seen[txt] = struct{}{}
			sr.dedup.mu.Unlock()
		}

		shouldInclude, matchedFilter, err = sr.op.applyFilter(txt)
		if err != nil {
			log.Logger.Warnw("error applying filter", "error", err)
			continue
		}
		if !shouldInclude {
			continue
		}

		var extractedTime time.Time
		scannedBytes := scanner.Bytes()

		if sr.op.extractTime != nil {
			parsedTime, extractedLine, err := sr.op.extractTime(scannedBytes)
			if err != nil {
				log.Logger.Errorw("error extracting time", "error", err)
			} else if len(extractedLine) > 0 {
				extractedTime = parsedTime.UTC()
				scannedBytes = extractedLine
			}
		}

		if extractedTime.IsZero() {
			extractedTime = time.Now().UTC()
		}

		if sr.op.ProcessMatched != nil {
			sr.op.ProcessMatched(extractedTime, scannedBytes, matchedFilter)
		}

		lineToSend := Line{
			Line: &tail.Line{
				Text: string(scannedBytes),
				Time: extractedTime,
			},
			MatchedFilter: matchedFilter,
		}

		select {
		case <-sr.ctx.Done():
			return

		case sr.lineC <- lineToSend:

		default:
			log.Logger.Warnw("channel is full -- dropped output", "pid", sr.proc.PID(), "labels", sr.proc.Labels())
		}
	}
}

func (sr *commandStreamer) waitCommand() {
	defer func() {
		close(sr.lineC)

		if sr.dedupEnabled {
			sr.dedup.mu.Lock()
			for k := range sr.dedup.seen {
				delete(sr.dedup.seen, k)
			}
			sr.dedup.mu.Unlock()
			seenPool.Put(sr.dedup)
		}

		if err := sr.proc.Close(sr.ctx); err != nil {
			log.Logger.Warnw("failed to abort command", "err", err)
		}
	}()

	select {
	case <-sr.ctx.Done():
	case <-sr.proc.Wait():
	}
}
