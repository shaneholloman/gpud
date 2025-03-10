package persistencemode

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leptonai/gpud/components"
	nvidia_query "github.com/leptonai/gpud/pkg/nvidia-query"
	nvidia_query_nvml "github.com/leptonai/gpud/pkg/nvidia-query/nvml"
)

// ToOutput converts nvidia_query.Output to Output.
// It returns an empty non-nil object, if the input or the required field is nil (e.g., i.SMI).
func ToOutput(i *nvidia_query.Output) *Output {
	if i == nil {
		return &Output{}
	}

	o := &Output{}

	if i.NVML != nil {
		for _, device := range i.NVML.DeviceInfos {
			o.PersistenceModesNVML = append(o.PersistenceModesNVML, device.PersistenceMode)
		}
	}

	return o
}

type Output struct {
	PersistenceModesNVML []nvidia_query_nvml.PersistenceMode `json:"persistence_modes_nvml"`
}

func (o *Output) JSON() ([]byte, error) {
	return json.Marshal(o)
}

const (
	StateNamePersistenceMode = "persistence_mode"

	StateKeyPersistenceModeData       = "data"
	StateKeyPersistenceModeEncoding   = "encoding"
	StateValueMemoryUsageEncodingJSON = "json"
)

// Returns the output evaluation reason and its healthy-ness.
func (o *Output) Evaluate() (string, bool, error) {
	reasons := []string{}

	enabled := true
	for _, p := range o.PersistenceModesNVML {
		// legacy mode (https://docs.nvidia.com/deploy/driver-persistence/index.html#installation)
		// "The reason why we cannot immediately deprecate the legacy persistence mode and switch transparently to the NVIDIA Persistence Daemon is because at this time,
		// we cannot guarantee that the NVIDIA Persistence Daemon will be running. This would be a feature regression as persistence mode might not be available out-of- the-box."
		if !p.Enabled {
			reasons = append(reasons, fmt.Sprintf("persistence mode is not enabled on %s (NVML)", p.UUID))
			enabled = false
		}
	}

	return strings.Join(reasons, "; "), enabled, nil
}

func (o *Output) States() ([]components.State, error) {
	outputReasons, healthy, err := o.Evaluate()
	if err != nil {
		return nil, err
	}
	b, _ := o.JSON()
	state := components.State{
		Name:    StateNamePersistenceMode,
		Healthy: healthy,
		Reason:  outputReasons,
		ExtraInfo: map[string]string{
			StateKeyPersistenceModeData:     string(b),
			StateKeyPersistenceModeEncoding: StateValueMemoryUsageEncodingJSON,
		},
	}
	return []components.State{state}, nil
}
