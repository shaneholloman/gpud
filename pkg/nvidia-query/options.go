package query

import (
	"database/sql"

	events_db "github.com/leptonai/gpud/pkg/events-db"
)

type Op struct {
	dbRW                  *sql.DB
	dbRO                  *sql.DB
	xidEventsStore        events_db.Store
	hwslowdownEventsStore events_db.Store
	nvidiaSMIQueryCommand string
	ibstatCommand         string
	debug                 bool
}

type OpOption func(*Op)

func (op *Op) applyOpts(opts []OpOption) error {
	for _, opt := range opts {
		opt(op)
	}

	if op.nvidiaSMIQueryCommand == "" {
		op.nvidiaSMIQueryCommand = "nvidia-smi --query"
	}
	if op.ibstatCommand == "" {
		op.ibstatCommand = "ibstat"
	}

	return nil
}

func WithDBRW(db *sql.DB) OpOption {
	return func(op *Op) {
		op.dbRW = db
	}
}

func WithDBRO(db *sql.DB) OpOption {
	return func(op *Op) {
		op.dbRO = db
	}
}

func WithXidEventsStore(store events_db.Store) OpOption {
	return func(op *Op) {
		op.xidEventsStore = store
	}
}

func WithHWSlowdownEventsStore(store events_db.Store) OpOption {
	return func(op *Op) {
		op.hwslowdownEventsStore = store
	}
}

func WithNvidiaSMIQueryCommand(p string) OpOption {
	return func(op *Op) {
		op.nvidiaSMIQueryCommand = p
	}
}

// Specifies the ibstat binary path to overwrite the default path.
func WithIbstatCommand(p string) OpOption {
	return func(op *Op) {
		op.ibstatCommand = p
	}
}

func WithDebug(debug bool) OpOption {
	return func(op *Op) {
		op.debug = debug
	}
}
