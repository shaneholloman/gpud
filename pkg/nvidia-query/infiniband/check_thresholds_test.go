package infiniband

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckPortsAndRate(t *testing.T) {
	tt := []struct {
		fileName               string
		expectedPhysicalStates []string
		expectedAtLeastRate    int
		expectedCount          int
		expectedPortNames      []string
	}{
		{
			fileName:               "testdata/ibstat.47.0.a100.all.active.0",
			expectedPhysicalStates: []string{"LinkUp"},
			expectedAtLeastRate:    200,
			expectedCount:          8,
			expectedPortNames:      []string{"mlx5_1", "mlx5_2", "mlx5_3", "mlx5_4", "mlx5_5", "mlx5_6", "mlx5_7", "mlx5_8"},
		},
		{
			fileName:               "testdata/ibstat.47.0.a100.all.active.0",
			expectedPhysicalStates: []string{"LinkUp"},
			expectedAtLeastRate:    100,
			expectedCount:          8,
			expectedPortNames:      []string{"mlx5_1", "mlx5_2", "mlx5_3", "mlx5_4", "mlx5_5", "mlx5_6", "mlx5_7", "mlx5_8"},
		},
		{
			fileName:               "testdata/ibstat.47.0.h100.all.active.0",
			expectedPhysicalStates: []string{"LinkUp"},
			expectedAtLeastRate:    400,
			expectedCount:          0,
			expectedPortNames:      []string{},
		},
		{
			fileName:               "testdata/ibstat.47.0.h100.all.active.1",
			expectedPhysicalStates: []string{"LinkUp"},
			expectedAtLeastRate:    400,
			expectedCount:          0,
			expectedPortNames:      []string{},
		},
		{
			fileName:               "testdata/ibstat.47.0.h100.all.active.2",
			expectedPhysicalStates: []string{"LinkUp"},
			expectedAtLeastRate:    400,
			expectedCount:          8,
			expectedPortNames:      []string{"mlx5_0", "mlx5_1", "mlx5_4", "mlx5_5", "mlx5_6", "mlx5_7", "mlx5_8", "mlx5_9"},
		},
		{
			fileName:               "testdata/ibstat.47.0.h100.some.down.0",
			expectedPhysicalStates: []string{"LinkUp"},
			expectedAtLeastRate:    400,
			expectedCount:          8,
			expectedPortNames:      []string{"mlx5_0", "mlx5_10", "mlx5_11", "mlx5_3", "mlx5_4", "mlx5_5", "mlx5_6", "mlx5_9"},
		},
		{
			fileName:               "testdata/ibstat.47.0.h100.some.down.1",
			expectedAtLeastRate:    400,
			expectedPhysicalStates: []string{"LinkUp"},
			expectedCount:          0,
			expectedPortNames:      []string{},
		},
		{
			fileName:               "testdata/ibstat.47.0.h100.some.down.with.polling.1",
			expectedPhysicalStates: []string{"Disabled", "Polling"},
			expectedAtLeastRate:    0,
			expectedCount:          0,
			expectedPortNames:      []string{},
		},
	}
	for _, tc := range tt {
		t.Run(tc.fileName, func(t *testing.T) {
			content, err := os.ReadFile(tc.fileName)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", tc.fileName, err)
			}
			parsed, err := ParseIBStat(string(content))
			if err != nil {
				t.Fatalf("Failed to parse ibstat file %s: %v", tc.fileName, err)
			}
			matched := checkPortsAndRate(
				parsed.IBPorts(),
				tc.expectedPhysicalStates,
				tc.expectedAtLeastRate,
			)
			if len(matched) != tc.expectedCount {
				t.Errorf("Expected %d cards, got %d", tc.expectedCount, len(matched))
			}
			// Extract device names from matched ports
			matchedNames := make([]string, 0, len(matched))
			for _, port := range matched {
				matchedNames = append(matchedNames, port.Device)
			}
			if !reflect.DeepEqual(matchedNames, tc.expectedPortNames) {
				t.Errorf("Expected %v, got %v", tc.expectedPortNames, matchedNames)
			}
		})
	}
}

func TestCheckPortsAndRateWithPhysicalState(t *testing.T) {
	ports := []IBPort{
		{
			Device:        "mlx5_0",
			State:         "Active",
			PhysicalState: "LinkUp",
			RateGBSec:     200,
			LinkLayer:     "Infiniband",
		},
		{
			Device:        "mlx5_1",
			State:         "Down",
			PhysicalState: "Disabled",
			RateGBSec:     200,
			LinkLayer:     "Infiniband",
		},
	}

	// Test with LinkUp physical state
	matchedLinkUp := checkPortsAndRate(ports, []string{"LinkUp"}, 200)
	assert.Equal(t, 1, len(matchedLinkUp), "Should match only the LinkUp physical state port")
	assert.Equal(t, "mlx5_0", matchedLinkUp[0].Device, "Device mlx5_0 should be matched")

	// Test with Disabled physical state
	matchedDisabled := checkPortsAndRate(ports, []string{"Disabled"}, 200)
	assert.Equal(t, 1, len(matchedDisabled), "Should match only the Disabled physical state port")
	assert.Equal(t, "mlx5_1", matchedDisabled[0].Device, "Device mlx5_1 should be matched")

	// Test with a physical state that doesn't match
	matchedNone := checkPortsAndRate(ports, []string{"Polling"}, 200)
	assert.Equal(t, 0, len(matchedNone), "Should not match any port")
}

func TestCheckPortsAndRate_IsIBPortFiltering(t *testing.T) {
	tests := []struct {
		name                   string
		ports                  []IBPort
		expectedPhysicalStates []string
		atLeastRate            int
		expectedMatchCount     int
		expectedDevices        []string
	}{
		{
			name: "mixed link layers - only infiniband should match",
			ports: []IBPort{
				{
					Device:        "mlx5_0",
					LinkLayer:     "Infiniband",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_1",
					LinkLayer:     "Ethernet",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_2",
					LinkLayer:     "INFINIBAND",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
			},
			expectedPhysicalStates: []string{"LinkUp"},
			atLeastRate:            400,
			expectedMatchCount:     2,
			expectedDevices:        []string{"mlx5_0", "mlx5_2"},
		},
		{
			name: "all ethernet ports - none should match",
			ports: []IBPort{
				{
					Device:        "eth0",
					LinkLayer:     "Ethernet",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "eth1",
					LinkLayer:     "ethernet",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
			},
			expectedPhysicalStates: []string{"LinkUp"},
			atLeastRate:            400,
			expectedMatchCount:     0,
			expectedDevices:        []string{},
		},
		{
			name: "all infiniband ports with case variations",
			ports: []IBPort{
				{
					Device:        "mlx5_0",
					LinkLayer:     "infiniband",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_1",
					LinkLayer:     "INFINIBAND",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_2",
					LinkLayer:     "InfiniBand",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_3",
					LinkLayer:     "Infiniband",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
			},
			expectedPhysicalStates: []string{"LinkUp"},
			atLeastRate:            400,
			expectedMatchCount:     4,
			expectedDevices:        []string{"mlx5_0", "mlx5_1", "mlx5_2", "mlx5_3"},
		},
		{
			name: "infiniband ports with different physical states - filtering by IsIBPort first",
			ports: []IBPort{
				{
					Device:        "mlx5_0",
					LinkLayer:     "Infiniband",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_1",
					LinkLayer:     "Ethernet",
					State:         "Active",
					PhysicalState: "LinkUp", // This would match physical state but should be filtered out by LinkLayer
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_2",
					LinkLayer:     "Infiniband",
					State:         "Down",
					PhysicalState: "Disabled",
					RateGBSec:     400,
				},
			},
			expectedPhysicalStates: []string{"LinkUp"},
			atLeastRate:            400,
			expectedMatchCount:     1,
			expectedDevices:        []string{"mlx5_0"},
		},
		{
			name: "infiniband ports with rate filtering - IsIBPort check first",
			ports: []IBPort{
				{
					Device:        "mlx5_0",
					LinkLayer:     "Infiniband",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     200, // Below threshold
				},
				{
					Device:        "mlx5_1",
					LinkLayer:     "Ethernet",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400, // Meets rate but wrong LinkLayer
				},
				{
					Device:        "mlx5_2",
					LinkLayer:     "Infiniband",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400, // Meets all criteria
				},
			},
			expectedPhysicalStates: []string{"LinkUp"},
			atLeastRate:            400,
			expectedMatchCount:     1,
			expectedDevices:        []string{"mlx5_2"},
		},
		{
			name: "empty link layer should not match",
			ports: []IBPort{
				{
					Device:        "mlx5_0",
					LinkLayer:     "",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_1",
					LinkLayer:     "Infiniband",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
			},
			expectedPhysicalStates: []string{"LinkUp"},
			atLeastRate:            400,
			expectedMatchCount:     1,
			expectedDevices:        []string{"mlx5_1"},
		},
		{
			name: "unknown link layer should not match",
			ports: []IBPort{
				{
					Device:        "mlx5_0",
					LinkLayer:     "Unknown",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_1",
					LinkLayer:     "SomeOtherProtocol",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_2",
					LinkLayer:     "Infiniband",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
			},
			expectedPhysicalStates: []string{"LinkUp"},
			atLeastRate:            400,
			expectedMatchCount:     1,
			expectedDevices:        []string{"mlx5_2"},
		},
		{
			name: "no physical state filter - only LinkLayer matters",
			ports: []IBPort{
				{
					Device:        "mlx5_0",
					LinkLayer:     "Infiniband",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_1",
					LinkLayer:     "Ethernet",
					State:         "Active",
					PhysicalState: "LinkUp",
					RateGBSec:     400,
				},
				{
					Device:        "mlx5_2",
					LinkLayer:     "Infiniband",
					State:         "Down",
					PhysicalState: "Disabled",
					RateGBSec:     400,
				},
			},
			expectedPhysicalStates: []string{}, // Empty means match all physical states
			atLeastRate:            400,
			expectedMatchCount:     2,
			expectedDevices:        []string{"mlx5_0", "mlx5_2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := checkPortsAndRate(tt.ports, tt.expectedPhysicalStates, tt.atLeastRate)

			assert.Equal(t, tt.expectedMatchCount, len(matched), "Number of matched ports should be correct")

			// Check that all matched devices are in the expected list
			matchedDevices := make([]string, len(matched))
			for i, port := range matched {
				matchedDevices[i] = port.Device
			}

			assert.ElementsMatch(t, tt.expectedDevices, matchedDevices, "Matched devices should match expected devices")

			// Verify that all matched ports are InfiniBand ports
			for _, port := range matched {
				assert.True(t, port.IsIBPort(), "All matched ports should be InfiniBand ports, but %s with LinkLayer %s was matched", port.Device, port.LinkLayer)
			}
		})
	}
}

func TestEvaluatePortsAndRate(t *testing.T) {
	tests := []struct {
		name                 string
		allPorts             []IBPort
		atLeastPorts         int
		atLeastRate          int
		expectError          bool
		expectedErrorMsg     string
		expectedProblemCount int
	}{
		{
			name: "all ports meet thresholds",
			allPorts: []IBPort{
				{Device: "mlx5_0", State: "Active", PhysicalState: "LinkUp", RateGBSec: 400, LinkLayer: "Infiniband"},
				{Device: "mlx5_1", State: "Active", PhysicalState: "LinkUp", RateGBSec: 400, LinkLayer: "Infiniband"},
			},
			atLeastPorts: 2,
			atLeastRate:  400,
			expectError:  false,
		},
		{
			name: "zero thresholds",
			allPorts: []IBPort{
				{Device: "mlx5_0", State: "Active", PhysicalState: "LinkUp", RateGBSec: 400, LinkLayer: "Infiniband"},
			},
			atLeastPorts: 0,
			atLeastRate:  0,
			expectError:  false,
		},
		{
			name: "insufficient ports with required rate",
			allPorts: []IBPort{
				{Device: "mlx5_0", State: "Active", PhysicalState: "LinkUp", RateGBSec: 200, LinkLayer: "Infiniband"},
				{Device: "mlx5_1", State: "Active", PhysicalState: "LinkUp", RateGBSec: 200, LinkLayer: "Infiniband"},
			},
			atLeastPorts:     2,
			atLeastRate:      400,
			expectError:      true,
			expectedErrorMsg: "only 0 port(s) are active and >=400 Gb/s, expect >=2 port(s)",
		},
		{
			name: "some ports disabled",
			allPorts: []IBPort{
				{Device: "mlx5_0", State: "Active", PhysicalState: "LinkUp", RateGBSec: 400, LinkLayer: "Infiniband"},
				{Device: "mlx5_1", State: "Down", PhysicalState: "Disabled", RateGBSec: 400, LinkLayer: "Infiniband"},
			},
			atLeastPorts:         2,
			atLeastRate:          400,
			expectError:          true,
			expectedErrorMsg:     "only 1 port(s) are active and >=400 Gb/s, expect >=2 port(s); 1 device(s) physical state Disabled (mlx5_1)",
			expectedProblemCount: 1,
		},
		{
			name: "some ports polling",
			allPorts: []IBPort{
				{Device: "mlx5_0", State: "Active", PhysicalState: "LinkUp", RateGBSec: 400, LinkLayer: "Infiniband"},
				{Device: "mlx5_1", State: "Init", PhysicalState: "Polling", RateGBSec: 400, LinkLayer: "Infiniband"},
			},
			atLeastPorts:         2,
			atLeastRate:          400,
			expectError:          true,
			expectedErrorMsg:     "only 1 port(s) are active and >=400 Gb/s, expect >=2 port(s); 1 device(s) physical state Polling (mlx5_1) -- connecton lost from this card to other cards/switches",
			expectedProblemCount: 1,
		},
		{
			name: "mixed disabled and polling",
			allPorts: []IBPort{
				{Device: "mlx5_0", State: "Active", PhysicalState: "LinkUp", RateGBSec: 400, LinkLayer: "Infiniband"},
				{Device: "mlx5_1", State: "Down", PhysicalState: "Disabled", RateGBSec: 400, LinkLayer: "Infiniband"},
				{Device: "mlx5_2", State: "Init", PhysicalState: "Polling", RateGBSec: 400, LinkLayer: "Infiniband"},
			},
			atLeastPorts:         3,
			atLeastRate:          400,
			expectError:          true,
			expectedErrorMsg:     "only 1 port(s) are active and >=400 Gb/s, expect >=3 port(s); 1 device(s) physical state Disabled (mlx5_1); 1 device(s) physical state Polling (mlx5_2) -- connecton lost from this card to other cards/switches",
			expectedProblemCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problematicPorts, err := EvaluatePortsAndRate(tt.allPorts, tt.atLeastPorts, tt.atLeastRate)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErrorMsg, err.Error())
				assert.Equal(t, tt.expectedProblemCount, len(problematicPorts))
			} else {
				assert.NoError(t, err)
				assert.Nil(t, problematicPorts)
			}
		})
	}
}

// Test with real testdata files
func TestEvaluatePortsAndRateWithTestdata(t *testing.T) {
	tests := []struct {
		fileName         string
		atLeastPorts     int
		atLeastRate      int
		expectError      bool
		problemPortCount int
	}{
		{
			fileName:     "testdata/ibstat.47.0.a100.all.active.0",
			atLeastPorts: 8,
			atLeastRate:  200,
			expectError:  false,
		},
		{
			fileName:         "testdata/ibstat.47.0.a100.all.active.0",
			atLeastPorts:     10,
			atLeastRate:      200,
			expectError:      true,
			problemPortCount: 0, // All IB ports are up, just not enough of them
		},
		{
			fileName:         "testdata/ibstat.47.0.h100.all.active.0",
			atLeastPorts:     1,
			atLeastRate:      400,
			expectError:      true,
			problemPortCount: 0, // All ports are Ethernet, so no IB ports match
		},
		{
			fileName:     "testdata/ibstat.47.0.h100.some.down.0",
			atLeastPorts: 8,
			atLeastRate:  400,
			expectError:  false,
		},
		{
			fileName:         "testdata/ibstat.47.0.h100.some.down.1",
			atLeastPorts:     1,
			atLeastRate:      400,
			expectError:      true,
			problemPortCount: 0, // All ports are Ethernet, so no IB ports match
		},
	}

	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			content, err := os.ReadFile(tt.fileName)
			require.NoError(t, err)

			parsed, err := ParseIBStat(string(content))
			require.NoError(t, err)

			problematicPorts, err := EvaluatePortsAndRate(parsed.IBPorts(), tt.atLeastPorts, tt.atLeastRate)

			if tt.expectError {
				assert.Error(t, err)
				if tt.problemPortCount > 0 {
					assert.Equal(t, tt.problemPortCount, len(problematicPorts))
				}
			} else {
				assert.NoError(t, err)
				assert.Nil(t, problematicPorts)
			}
		})
	}
}

func TestIBPort_IsIBPort(t *testing.T) {
	tests := []struct {
		name      string
		linkLayer string
		expected  bool
	}{
		{
			name:      "infiniband lowercase",
			linkLayer: "infiniband",
			expected:  true,
		},
		{
			name:      "infiniband uppercase",
			linkLayer: "INFINIBAND",
			expected:  true,
		},
		{
			name:      "infiniband capitalized",
			linkLayer: "Infiniband",
			expected:  true,
		},
		{
			name:      "infiniband mixed case",
			linkLayer: "InfiniBand",
			expected:  true,
		},
		{
			name:      "infiniband with extra spaces - trimmed input",
			linkLayer: "InfiniBand",
			expected:  true,
		},
		{
			name:      "ethernet lowercase",
			linkLayer: "ethernet",
			expected:  false,
		},
		{
			name:      "ethernet capitalized",
			linkLayer: "Ethernet",
			expected:  false,
		},
		{
			name:      "ethernet uppercase",
			linkLayer: "ETHERNET",
			expected:  false,
		},
		{
			name:      "empty string",
			linkLayer: "",
			expected:  false,
		},
		{
			name:      "random string",
			linkLayer: "random",
			expected:  false,
		},
		{
			name:      "partial match",
			linkLayer: "infini",
			expected:  false,
		},
		{
			name:      "contains infiniband but not exact",
			linkLayer: "infiniband_extra",
			expected:  false,
		},
		{
			name:      "whitespace only",
			linkLayer: "   ",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := IBPort{
				LinkLayer: tt.linkLayer,
			}
			result := port.IsIBPort()
			assert.Equal(t, tt.expected, result, "Expected IsIBPort() to return %v for LinkLayer %q", tt.expected, tt.linkLayer)
		})
	}
}
