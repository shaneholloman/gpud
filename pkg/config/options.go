package config

import (
	pkgconfigcommon "github.com/leptonai/gpud/pkg/config/common"
	infinibandclass "github.com/leptonai/gpud/pkg/nvidia-query/infiniband/class"
)

type Op struct {
	pkgconfigcommon.ToolOverwrites
}

type OpOption func(*Op)

func (op *Op) ApplyOpts(opts []OpOption) error {
	for _, opt := range opts {
		opt(op)
	}

	if op.InfinibandClassRootDir == "" {
		op.InfinibandClassRootDir = infinibandclass.DefaultRootDir
	}

	return nil
}

// Specifies the root directory of the InfiniBand class.
func WithInfinibandClassRootDir(p string) OpOption {
	return func(op *Op) {
		op.InfinibandClassRootDir = p
	}
}
