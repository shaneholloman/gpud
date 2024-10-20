package config

type Op struct {
	FilesToCheck                  []string
	DockerIgnoreConnectionErrors  bool
	KubeletIgnoreConnectionErrors bool
}

type OpOption func(*Op)

func (op *Op) ApplyOpts(opts []OpOption) error {
	for _, opt := range opts {
		opt(op)
	}

	return nil
}

func WithFilesToCheck(files ...string) OpOption {
	return func(op *Op) {
		op.FilesToCheck = append(op.FilesToCheck, files...)
	}
}

func WithDockerIgnoreConnectionErrors(b bool) OpOption {
	return func(op *Op) {
		op.DockerIgnoreConnectionErrors = b
	}
}

func WithKubeletIgnoreConnectionErrors(b bool) OpOption {
	return func(op *Op) {
		op.KubeletIgnoreConnectionErrors = b
	}
}
