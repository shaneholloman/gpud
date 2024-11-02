package xid

import (
	"encoding/json"
	"reflect"
	"testing"

	nvidia_query_nvml "github.com/leptonai/gpud/components/accelerator/nvidia/query/nvml"
	nvidia_query_xid "github.com/leptonai/gpud/components/accelerator/nvidia/query/xid"
)

func TestOutputEvaluate(t *testing.T) {
	tests := []struct {
		name             string
		input            *Output
		onlyGPUdCritical bool
		wantReason       Reason
		wantHealthy      bool
		wantErr          bool
	}{
		{
			name:             "no errors",
			input:            &Output{},
			onlyGPUdCritical: false,
			wantReason: Reason{
				Messages: []string{"no xid error found"},
			},
			wantHealthy: true,
			wantErr:     false,
		},
		{
			name: "nvml xid error",
			input: &Output{
				NVMLXidEvent: &nvidia_query_nvml.XidEvent{
					Xid:                          79,
					XidCriticalErrorMarkedByNVML: true,
					XidCriticalErrorMarkedByGPUd: true,
					Detail: &nvidia_query_xid.Detail{
						Description: "GPU has fallen off the bus",
					},
				},
			},
			onlyGPUdCritical: false,
			wantReason: Reason{
				Messages: []string{"nvml xid 79 event (critical)"},
				Errors: map[uint64]XidError{
					79: {
						DataSource:                   "nvml",
						Xid:                          79,
						XidDescription:               "GPU has fallen off the bus",
						XidCriticalErrorMarkedByNVML: true,
						XidCriticalErrorMarkedByGPUd: true,
					},
				},
			},
			wantHealthy: false,
			wantErr:     false,
		},
		{
			name: "dmesg xid error",
			input: &Output{
				DmesgErrors: []nvidia_query_xid.DmesgError{
					{
						DetailFound: true,
						Detail: &nvidia_query_xid.Detail{
							XID:                       79,
							CriticalErrorMarkedByGPUd: true,
						},
					},
				},
			},
			onlyGPUdCritical: false,
			wantReason: Reason{
				Messages: []string{"dmesg xid 79 event (critical)"},
				Errors: map[uint64]XidError{
					79: {
						DataSource:                   "dmesg",
						Xid:                          79,
						XidCriticalErrorMarkedByGPUd: true,
					},
				},
			},
			wantHealthy: false,
			wantErr:     false,
		},
		{
			name: "both nvml and dmesg errors",
			input: &Output{
				NVMLXidEvent: &nvidia_query_nvml.XidEvent{
					Xid:                          79,
					XidCriticalErrorMarkedByNVML: true,
					XidCriticalErrorMarkedByGPUd: true,
				},
				DmesgErrors: []nvidia_query_xid.DmesgError{
					{
						DetailFound: true,
						Detail: &nvidia_query_xid.Detail{
							XID:                       79,
							CriticalErrorMarkedByGPUd: true,
						},
					},
					{
						DetailFound: true,
						Detail: &nvidia_query_xid.Detail{
							XID:                       80,
							CriticalErrorMarkedByGPUd: true,
						},
					},
				},
			},
			onlyGPUdCritical: false,
			wantReason: Reason{
				Messages: []string{"nvml xid 79 event (critical)", "dmesg xid 80 event (critical)"},
				Errors: map[uint64]XidError{
					79: {
						DataSource:                   "nvml",
						Xid:                          79,
						XidCriticalErrorMarkedByNVML: true,
						XidCriticalErrorMarkedByGPUd: true,
					},
					80: {
						DataSource:                   "dmesg",
						Xid:                          80,
						XidCriticalErrorMarkedByGPUd: true,
					},
				},
			},
			wantHealthy: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReason, gotHealthy, err := tt.input.Evaluate(tt.onlyGPUdCritical)
			if (err != nil) != tt.wantErr {
				t.Errorf("Output.Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare the actual structs instead of JSON representations
			if !reflect.DeepEqual(gotReason, tt.wantReason) {
				// For better error messages, still use JSON for output
				gotJSON, _ := json.MarshalIndent(gotReason, "", "  ")
				wantJSON, _ := json.MarshalIndent(tt.wantReason, "", "  ")
				t.Errorf("Output.Evaluate() reason = \n%s\n\nwant\n%s", string(gotJSON), string(wantJSON))
			}

			if gotHealthy != tt.wantHealthy {
				t.Errorf("Output.Evaluate() healthy = %v, want %v", gotHealthy, tt.wantHealthy)
			}
		})
	}
}