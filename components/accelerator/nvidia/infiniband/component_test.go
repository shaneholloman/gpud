package infiniband

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leptonai/gpud/components"
	nvidia_common "github.com/leptonai/gpud/pkg/config/common"
	"github.com/leptonai/gpud/pkg/nvidia-query/infiniband"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name        string
		output      *infiniband.IbstatOutput
		config      infiniband.ExpectedPortStates
		wantReason  string
		wantHealthy bool
		wantErr     bool
	}{
		{
			name:   "thresholds not set",
			output: &infiniband.IbstatOutput{},
			config: infiniband.ExpectedPortStates{
				AtLeastPorts: 0,
				AtLeastRate:  0,
			},
			wantReason:  msgThresholdNotSetSkipped,
			wantHealthy: true,
			wantErr:     false,
		},
		{
			name: "healthy state with matching ports and rate",
			output: &infiniband.IbstatOutput{
				Raw: "",
				Parsed: infiniband.IBStatCards{
					{
						Name: "mlx5_0",
						Port1: infiniband.IBStatPort{
							State:         "Active",
							PhysicalState: "LinkUp",
							Rate:          200,
						},
					},
					{
						Name: "mlx5_1",
						Port1: infiniband.IBStatPort{
							State:         "Active",
							PhysicalState: "LinkUp",
							Rate:          200,
						},
					},
				},
			},
			config: infiniband.ExpectedPortStates{
				AtLeastPorts: 2,
				AtLeastRate:  200,
			},
			wantReason:  msgNoIbIssueFound,
			wantHealthy: true,
			wantErr:     false,
		},
		{
			name: "unhealthy state - not enough ports",
			output: &infiniband.IbstatOutput{
				Raw: "",
				Parsed: infiniband.IBStatCards{
					{
						Name: "mlx5_0",
						Port1: infiniband.IBStatPort{
							State:         "Active",
							PhysicalState: "LinkUp",
							Rate:          200,
						},
					},
				},
			},
			config: infiniband.ExpectedPortStates{
				AtLeastPorts: 2,
				AtLeastRate:  200,
			},
			wantReason:  "not enough LinkUp ports, only 1 LinkUp out of 1, expected at least 2 ports and 200 Gb/sec rate; some ports must be missing",
			wantHealthy: false,
			wantErr:     false,
		},
		{
			name: "unhealthy state - rate too low",
			output: &infiniband.IbstatOutput{
				Raw: "",
				Parsed: infiniband.IBStatCards{
					{
						Name: "mlx5_0",
						Port1: infiniband.IBStatPort{
							State:         "Active",
							PhysicalState: "LinkUp",
							Rate:          100,
						},
					},
					{
						Name: "mlx5_1",
						Port1: infiniband.IBStatPort{
							State:         "Active",
							PhysicalState: "LinkUp",
							Rate:          100,
						},
					},
				},
			},
			config: infiniband.ExpectedPortStates{
				AtLeastPorts: 2,
				AtLeastRate:  200,
			},
			wantReason:  "not enough LinkUp ports, only 0 LinkUp out of 2, expected at least 2 ports and 200 Gb/sec rate; some ports must be missing",
			wantHealthy: false,
			wantErr:     false,
		},
		{
			name: "unhealthy state - disabled ports",
			output: &infiniband.IbstatOutput{
				Raw: "",
				Parsed: infiniband.IBStatCards{
					{
						Name: "mlx5_0",
						Port1: infiniband.IBStatPort{
							State:         "Down",
							PhysicalState: "Disabled",
							Rate:          200,
						},
					},
					{
						Name: "mlx5_1",
						Port1: infiniband.IBStatPort{
							State:         "Down",
							PhysicalState: "Disabled",
							Rate:          200,
						},
					},
				},
			},
			config: infiniband.ExpectedPortStates{
				AtLeastPorts: 2,
				AtLeastRate:  200,
			},
			wantReason:  "not enough LinkUp ports, only 0 LinkUp out of 2, expected at least 2 ports and 200 Gb/sec rate; some ports might be down, 2 Disabled devices with Rate > 200 found (mlx5_0, mlx5_1)",
			wantHealthy: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, healthy, err := evaluate(tt.output, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantReason, reason)
			assert.Equal(t, tt.wantHealthy, healthy)
		})
	}
}

func TestDefaultExpectedPortStates(t *testing.T) {
	// Test default values
	defaults := GetDefaultExpectedPortStates()
	assert.Equal(t, 0, defaults.AtLeastPorts)
	assert.Equal(t, 0, defaults.AtLeastRate)

	// Test setting new values
	newStates := infiniband.ExpectedPortStates{
		AtLeastPorts: 2,
		AtLeastRate:  200,
	}
	SetDefaultExpectedPortStates(newStates)

	updated := GetDefaultExpectedPortStates()
	assert.Equal(t, newStates.AtLeastPorts, updated.AtLeastPorts)
	assert.Equal(t, newStates.AtLeastRate, updated.AtLeastRate)
}

func TestEvaluateWithTestData(t *testing.T) {
	// Read the test data file
	testDataPath := filepath.Join("testdata", "ibstat.47.0.h100.all.active.1")
	content, err := os.ReadFile(testDataPath)
	require.NoError(t, err, "Failed to read test data file")

	// Parse the test data
	cards, err := infiniband.ParseIBStat(string(content))
	require.NoError(t, err, "Failed to parse ibstat output")

	output := &infiniband.IbstatOutput{
		Raw:    string(content),
		Parsed: cards,
	}

	tests := []struct {
		name        string
		config      infiniband.ExpectedPortStates
		wantReason  string
		wantHealthy bool
		wantErr     bool
	}{
		{
			name: "healthy state - all H100 ports active at 400Gb/s",
			config: infiniband.ExpectedPortStates{
				AtLeastPorts: 8,   // Number of 400Gb/s ports in the test data
				AtLeastRate:  400, // Expected rate for H100 cards
			},
			wantReason:  msgNoIbIssueFound,
			wantHealthy: true,
			wantErr:     false,
		},
		{
			name: "healthy state - mixed rate ports",
			config: infiniband.ExpectedPortStates{
				AtLeastPorts: 12,  // Total number of ports in test data
				AtLeastRate:  100, // Minimum rate that includes all ports
			},
			wantReason:  msgNoIbIssueFound,
			wantHealthy: true,
			wantErr:     false,
		},
		{
			name: "unhealthy state - not enough high-rate ports",
			config: infiniband.ExpectedPortStates{
				AtLeastPorts: 12,  // Total number of ports
				AtLeastRate:  400, // Only 8 ports have this rate
			},
			wantReason:  "not enough LinkUp ports, only 8 LinkUp out of 12, expected at least 12 ports and 400 Gb/sec rate; some ports must be missing",
			wantHealthy: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, healthy, err := evaluate(output, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantReason, reason)
			assert.Equal(t, tt.wantHealthy, healthy)
		})
	}
}

func TestComponentStatesWithTestData(t *testing.T) {
	c := &component{
		toolOverwrites: nvidia_common.ToolOverwrites{
			IbstatCommand: "cat " + filepath.Join("testdata", "ibstat.47.0.h100.all.active.1"),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	states, err := c.getStates(ctx, infiniband.ExpectedPortStates{
		AtLeastPorts: 8,   // Number of 400Gb/s ports in the test data
		AtLeastRate:  400, // Expected rate for H100 cards
	})
	require.NoError(t, err)
	require.Len(t, states, 1)

	state := states[0]
	assert.Equal(t, "ibstat", state.Name)
	assert.True(t, state.Healthy)
	assert.Equal(t, components.StateHealthy, state.Health)
	assert.Equal(t, msgNoIbIssueFound, state.Reason)
	assert.Nil(t, state.SuggestedActions)
}

func TestComponentGetStatesWithThresholds(t *testing.T) {
	tests := []struct {
		name       string
		thresholds infiniband.ExpectedPortStates
		wantState  components.State
		wantErr    bool
	}{
		{
			name: "thresholds not set - should skip check",
			thresholds: infiniband.ExpectedPortStates{
				AtLeastPorts: 0,
				AtLeastRate:  0,
			},
			wantState: components.State{
				Name:   "ibstat",
				Health: components.StateHealthy,
				Reason: msgThresholdNotSetSkipped,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &component{}
			states, err := c.getStates(context.Background(), tt.thresholds)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, states, 1)
			assert.Equal(t, tt.wantState, states[0])
		})
	}
}

func TestComponentStatesNoIbstatCommand(t *testing.T) {
	testCases := []struct {
		name          string
		ibstatCommand string
	}{
		{
			name:          "empty command",
			ibstatCommand: "",
		},
		{
			name:          "non-existent command",
			ibstatCommand: "/non/existent/path/to/ibstat",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &component{
				toolOverwrites: nvidia_common.ToolOverwrites{
					IbstatCommand: tc.ibstatCommand,
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			states, err := c.getStates(ctx, infiniband.ExpectedPortStates{
				AtLeastPorts: 1,
				AtLeastRate:  100,
			})

			require.NoError(t, err)
			require.Len(t, states, 1)

			state := states[0]
			assert.Equal(t, "ibstat", state.Name)
			assert.Equal(t, components.StateUnhealthy, state.Health)
			assert.Contains(t, state.Reason, "ibstat threshold set but ibstat not found")
		})
	}
}
