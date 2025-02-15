// Package v1 provides the gpud v1 client for the server.
package v1

import (
	"github.com/leptonai/gpud/pkg/server"
)

type Op struct {
	requestContentType    string
	requestAcceptEncoding string
	components            map[string]any
}

type OpOption func(*Op)

func (op *Op) applyOpts(opts []OpOption) error {
	for _, opt := range opts {
		opt(op)
	}

	return nil
}

// WithRequestContentTypeYAML sets the request content type to YAML.
func WithRequestContentTypeYAML() OpOption {
	return func(op *Op) {
		op.requestContentType = server.RequestHeaderYAML
	}
}

// WithRequestContentTypeJSON sets the request content type to JSON.
func WithRequestContentTypeJSON() OpOption {
	return func(op *Op) {
		op.requestContentType = server.RequestHeaderJSON
	}
}

// WithAcceptEncodingGzip requests gzip encoding for the response.
func WithAcceptEncodingGzip() OpOption {
	return func(op *Op) {
		op.requestAcceptEncoding = server.RequestHeaderEncodingGzip
	}
}

func WithComponent(component string) OpOption {
	return func(op *Op) {
		if op.components == nil {
			op.components = make(map[string]any)
		}
		op.components[component] = nil
	}
}
