package accelerator

import (
	"context"

	"github.com/leptonai/gpud/pkg/file"
	nvidia_query "github.com/leptonai/gpud/pkg/nvidia-query"
)

type Type string

const (
	TypeUnknown Type = "unknown"
	TypeNVIDIA  Type = "nvidia"
)

// Returns the GPU type (e.g., "NVIDIA") and product name (e.g., "A100")
func DetectTypeAndProductName(ctx context.Context) (Type, string, error) {
	if _, err := file.LocateExecutable("nvidia-smi"); err == nil {
		productName, err := nvidia_query.LoadGPUDeviceName(ctx)
		if err != nil {
			return TypeNVIDIA, "unknown", err
		}
		return TypeNVIDIA, productName, nil
	}

	return TypeUnknown, "unknown", nil
}
