package remappedrows

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/leptonai/gpud/components"
	"github.com/leptonai/gpud/pkg/common"
	"github.com/leptonai/gpud/pkg/log"
	nvidia_query "github.com/leptonai/gpud/pkg/nvidia-query"
	nvidia_query_nvml "github.com/leptonai/gpud/pkg/nvidia-query/nvml"
)

func ToOutput(i *nvidia_query.Output) *Output {
	if i == nil {
		return nil
	}

	o := &Output{
		GPUProductName:                    i.GPUProductName(),
		MemoryErrorManagementCapabilities: i.MemoryErrorManagementCapabilities,
	}

	rmaMsgs := make([]string, 0)
	needRebootMsgs := make([]string, 0)

	if i.NVML != nil {
		for _, device := range i.NVML.DeviceInfos {
			o.RemappedRowsNVML = append(o.RemappedRowsNVML, device.RemappedRows)

			requiresReset := device.RemappedRows.RequiresReset()
			if requiresReset {
				msg := fmt.Sprintf("NVML indicates GPU %s needs reset (pending remapping %v)", device.UUID, requiresReset)
				needRebootMsgs = append(needRebootMsgs, msg)
			}

			rma := device.RemappedRows.QualifiesForRMA()
			if rma {
				msg := fmt.Sprintf("NVML indicates GPU %s qualifies for RMA (remapping failure occurred %v)", device.UUID, device.RemappedRows.RemappingFailed)
				rmaMsgs = append(rmaMsgs, msg)
			}
		}
	}

	if i.SMI != nil {
		for _, g := range i.SMI.GPUs {
			if g.RemappedRows == nil {
				continue
			}
			parsed, err := g.RemappedRows.Parse()
			if err != nil {
				log.Logger.Warnw("failed to parse temperature", "error", err)
				continue
			}
			o.RemappedRowsSMI = append(o.RemappedRowsSMI, parsed)

			requiresReset, err := parsed.RequiresReset()
			if err != nil {
				log.Logger.Warnw("failed to determine if GPU needs reset", "error", err)
				continue
			}
			if requiresReset {
				msg := fmt.Sprintf("nvidia-smi indicates GPU %q needs reset (pending remapping %v)", parsed.ID, requiresReset)
				needRebootMsgs = append(needRebootMsgs, msg)
			}

			rma, err := parsed.QualifiesForRMA()
			if err != nil {
				log.Logger.Warnw("failed to determine if GPU qualifies for RMA", "error", err)
				continue
			}
			if rma {
				msg := fmt.Sprintf("nvidia-smi indicates GPU %q qualifies for RMA (remapping failure occurred %v, remapped due to uncorrectable errors %s)", parsed.ID, parsed.RemappingFailed, parsed.RemappedDueToUncorrectableErrors)
				rmaMsgs = append(rmaMsgs, msg)
			}
		}
	}

	if len(needRebootMsgs) > 0 {
		if o.SuggestedActions == nil {
			o.SuggestedActions = &common.SuggestedActions{}
		}

		o.SuggestedActions.Descriptions = append(o.SuggestedActions.Descriptions, strings.Join(needRebootMsgs, ", "))
		o.SuggestedActions.RepairActions = append(o.SuggestedActions.RepairActions, common.RepairActionTypeRebootSystem)
	}
	if len(rmaMsgs) > 0 {
		if o.SuggestedActions == nil {
			o.SuggestedActions = &common.SuggestedActions{}
		}

		o.SuggestedActions.Descriptions = append(o.SuggestedActions.Descriptions, strings.Join(rmaMsgs, ", "))
		o.SuggestedActions.RepairActions = append(o.SuggestedActions.RepairActions, common.RepairActionTypeHardwareInspection)
	}

	return o
}

type Output struct {
	GPUProductName                    string                                         `json:"gpu_product_name"`
	MemoryErrorManagementCapabilities nvidia_query.MemoryErrorManagementCapabilities `json:"memory_error_management_capabilities"`
	RemappedRowsSMI                   []nvidia_query.ParsedSMIRemappedRows           `json:"remapped_rows_smi"`
	RemappedRowsNVML                  []nvidia_query_nvml.RemappedRows               `json:"remapped_rows_nvml"`

	// Recommended course of actions for any of the GPUs with a known issue.
	// For individual GPU details, see each per-GPU states.
	SuggestedActions *common.SuggestedActions `json:"suggested_actions,omitempty"`
}

func (o *Output) JSON() ([]byte, error) {
	return json.Marshal(o)
}

func ParseOutputJSON(data []byte) (*Output, error) {
	o := new(Output)
	if err := json.Unmarshal(data, o); err != nil {
		return nil, err
	}
	return o, nil
}

const (
	StateNameRemappedRows = "remapped_rows"

	StateKeyRemappedRowsData           = "data"
	StateKeyRemappedRowsEncoding       = "encoding"
	StateValueRemappedRowsEncodingJSON = "json"
)

func ParseStateRemappedRows(m map[string]string) (*Output, error) {
	data := m[StateKeyRemappedRowsData]
	return ParseOutputJSON([]byte(data))
}

func ParseStatesToOutput(states ...components.State) (*Output, error) {
	for _, state := range states {
		switch state.Name {
		case StateNameRemappedRows:
			o, err := ParseStateRemappedRows(state.ExtraInfo)
			if err != nil {
				return nil, err
			}
			return o, nil

		default:
			return nil, fmt.Errorf("unknown state name: %s", state.Name)
		}
	}
	return nil, errors.New("no state found")
}

func (o *Output) isRowRemappingSupported() bool {
	// even for "NVIDIA GeForce RTX 4090", this returns no error
	// thus "RemappedRowsNVML.Supported" is not a reliable way to check if row remapping is supported
	return o.MemoryErrorManagementCapabilities.RowRemapping
}

// Returns the output evaluation reason and its healthy-ness.
func (o *Output) Evaluate() (string, bool, error) {
	if o == nil {
		return "no data", true, nil
	}

	healthy := true
	reasons := []string{}

	if !o.isRowRemappingSupported() {
		reasons = append(reasons, fmt.Sprintf("GPU product name %q does not support row remapping (message: %q)", o.GPUProductName, o.MemoryErrorManagementCapabilities.Message))
	} else {
		for _, r := range o.RemappedRowsSMI {
			rma, err := r.QualifiesForRMA()
			if err != nil {
				healthy = false
				reasons = append(reasons, fmt.Sprintf("nvidia-smi GPU %s failed to determine if it qualifies for RMA: %s", r.ID, err.Error()))
				continue
			}
			if rma {
				healthy = false
				reasons = append(reasons, fmt.Sprintf("nvidia-smi GPU %s qualifies for RMA (remapping failure occurred %v, remapped due to uncorrectable errors %s)", r.ID, r.RemappingFailed, r.RemappedDueToUncorrectableErrors))
			}

			needsReset, err := r.RequiresReset()
			if err != nil {
				reasons = append(reasons, fmt.Sprintf("nvidia-smi GPU %s failed to determine if it needs reset: %s", r.ID, err.Error()))
				continue
			}
			if needsReset {
				healthy = false
				reasons = append(reasons, fmt.Sprintf("nvidia-smi GPU %s needs reset (pending remapping %v)", r.ID, needsReset))
			}
		}

		for _, r := range o.RemappedRowsNVML {
			if r.QualifiesForRMA() {
				healthy = false
				reasons = append(reasons, fmt.Sprintf("nvml GPU %s qualifies for RMA (remapping failure occurred %v, remapped due to uncorrectable errors %d)", r.UUID, r.RemappingFailed, r.RemappedDueToUncorrectableErrors))
			}
			if r.RequiresReset() {
				healthy = false
				reasons = append(reasons, fmt.Sprintf("nvml GPU %s needs reset (pending remapping %v)", r.UUID, r.RemappingPending))
			}
		}

		if len(reasons) == 0 {
			reasons = append(reasons, "no issue detected")
		}
	}

	reason := strings.Join(reasons, ", ")
	return reason, healthy, nil
}

func (o *Output) States() ([]components.State, error) {
	outputReasons, healthy, err := o.Evaluate()
	if err != nil {
		return nil, err
	}

	b, _ := o.JSON()
	state := components.State{
		Name:    StateNameRemappedRows,
		Healthy: healthy,
		Reason:  outputReasons,
		ExtraInfo: map[string]string{
			StateKeyRemappedRowsData:     string(b),
			StateKeyRemappedRowsEncoding: StateValueRemappedRowsEncodingJSON,
		},
	}

	if o.SuggestedActions != nil {
		state.SuggestedActions = o.SuggestedActions
	}

	return []components.State{state}, nil
}
