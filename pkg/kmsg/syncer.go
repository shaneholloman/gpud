package kmsg

import (
	"context"
	"time"

	apiv1 "github.com/leptonai/gpud/api/v1"
	"github.com/leptonai/gpud/pkg/eventstore"
	"github.com/leptonai/gpud/pkg/log"
)

// Syncer syncs kernel message matched by MatchFunc to eventstore bucket
type Syncer struct {
	ctx         context.Context
	watcher     Watcher
	matchFunc   MatchFunc
	eventBucket eventstore.Bucket
}

type MatchFunc func(line string) (eventName string, message string)

func NewSyncer(ctx context.Context, matchFunc MatchFunc, eventBucket eventstore.Bucket) (*Syncer, error) {
	return newSyncer(ctx, nil, matchFunc, eventBucket)
}

func newSyncer(ctx context.Context, watcher Watcher, matchFunc MatchFunc, eventBucket eventstore.Bucket) (*Syncer, error) {
	if watcher == nil {
		var err error
		watcher, err = NewWatcher()
		if err != nil {
			return nil, err
		}
	}

	w := &Syncer{
		ctx:         ctx,
		watcher:     watcher,
		matchFunc:   matchFunc,
		eventBucket: eventBucket,
	}
	ch, err := w.watcher.Watch()
	if err != nil {
		return nil, err
	}
	go w.sync(ch)
	return w, nil
}

func (w *Syncer) sync(ch <-chan Message) {
	for {
		select {
		case <-w.ctx.Done():
			return
		case kmsg, ok := <-ch:
			if !ok {
				return
			}

			name, message := w.matchFunc(kmsg.Message)
			if name == "" {
				continue
			}
			event := eventstore.Event{
				Time:    kmsg.Timestamp.UTC(),
				Name:    name,
				Message: message,
				Type:    string(apiv1.EventTypeWarning),
			}

			// lookup to prevent duplicate event insertions
			cctx, ccancel := context.WithTimeout(w.ctx, 15*time.Second)
			sameEvent, err := w.eventBucket.Find(cctx, event)
			ccancel()
			if err != nil {
				log.Logger.Errorw("failed to find event", "eventName", event.Name, "eventType", event.Type, "error", err)
			}
			if sameEvent != nil {
				continue
			}

			// insert event
			cctx, ccancel = context.WithTimeout(w.ctx, 15*time.Second)
			err = w.eventBucket.Insert(cctx, event)
			ccancel()
			if err != nil {
				log.Logger.Errorw("failed to insert event", "error", err)
			} else {
				log.Logger.Infow("successfully inserted event", "event", event.Name)
			}
		}
	}
}

func (w *Syncer) Close() {
	_ = w.watcher.Close()
}
