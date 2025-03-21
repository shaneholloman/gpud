package power

import (
	"encoding/json"
	"fmt"

	"github.com/leptonai/gpud/components"
	nvidia_query "github.com/leptonai/gpud/pkg/nvidia-query"
	nvidia_query_nvml "github.com/leptonai/gpud/pkg/nvidia-query/nvml"

	"sigs.k8s.io/yaml"
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
			o.UsagesNVML = append(o.UsagesNVML, device.Power)
		}
	}

	return o
}

type Output struct {
	UsagesNVML []nvidia_query_nvml.Power `json:"usages_nvml"`
}

func (o *Output) JSON() ([]byte, error) {
	return json.Marshal(o)
}

const (
	StateNamePowerUsage = "power_usage"

	StateKeyPowerUsageData           = "data"
	StateKeyPowerUsageEncoding       = "encoding"
	StateValuePowerUsageEncodingJSON = "json"
)

// Returns the output evaluation reason and its healthy-ness.
func (o *Output) Evaluate() (string, bool, error) {
	type temp struct {
		UUID        string `json:"uuid"`
		LimitW      string `json:"limit_w"`
		UsageW      string `json:"usage_w"`
		UsedPercent string `json:"used_percent"`
	}
	pows := make([]temp, len(o.UsagesNVML))
	for i, u := range o.UsagesNVML {
		pows[i] = temp{
			UUID:        u.UUID,
			LimitW:      fmt.Sprintf("%.2f W", float64(u.EnforcedLimitMilliWatts)/1000.0),
			UsageW:      fmt.Sprintf("%.2f W", float64(u.UsageMilliWatts)/1000.0),
			UsedPercent: u.UsedPercent,
		}
	}
	yb, err := yaml.Marshal(pows)
	if err != nil {
		return "", false, err
	}
	return string(yb), true, nil
}

func (o *Output) States() ([]components.State, error) {
	outputReasons, healthy, err := o.Evaluate()
	if err != nil {
		return nil, err
	}
	b, _ := o.JSON()
	state := components.State{
		Name:    StateNamePowerUsage,
		Healthy: healthy,
		Reason:  outputReasons,
		ExtraInfo: map[string]string{
			StateKeyPowerUsageData:     string(b),
			StateKeyPowerUsageEncoding: StateValuePowerUsageEncodingJSON,
		},
	}
	return []components.State{state}, nil
}
