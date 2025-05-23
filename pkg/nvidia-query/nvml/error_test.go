package nvml

import (
	"errors"
	"testing"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/stretchr/testify/assert"
)

func TestIsNotSupportError(t *testing.T) {
	tests := []struct {
		name     string
		ret      nvml.Return
		expected bool
	}{
		{
			name:     "Direct ERROR_NOT_SUPPORTED match",
			ret:      nvml.ERROR_NOT_SUPPORTED,
			expected: true,
		},
		{
			name:     "Success is not a not-supported error",
			ret:      nvml.SUCCESS,
			expected: false,
		},
		{
			name:     "Unknown error is not a not-supported error",
			ret:      nvml.ERROR_UNKNOWN,
			expected: false,
		},
		{
			name:     "Version mismatch error is not a not-supported error",
			ret:      nvml.ERROR_ARGUMENT_VERSION_MISMATCH,
			expected: false,
		},
	}

	// Override nvml.ErrorString for testing string-based matches
	originalErrorString := nvml.ErrorString
	defer func() {
		nvml.ErrorString = originalErrorString
	}()

	nvml.ErrorString = func(ret nvml.Return) string {
		switch ret {
		case nvml.Return(1000):
			return "operation is not supported on this device"
		case nvml.Return(1001):
			return "THIS OPERATION IS NOT SUPPORTED"
		case nvml.Return(1002):
			return "Feature Not Supported"
		case nvml.Return(1003):
			return "  not supported  "
		case nvml.Return(1004):
			return "The requested operation is not supported on device 0"
		case nvml.Return(1005):
			return "Some other error"
		case nvml.Return(1006):
			return ""
		case nvml.Return(1007):
			return "notsupported" // No space between 'not' and 'supported'
		default:
			return originalErrorString(ret)
		}
	}

	// Add string-based test cases
	stringBasedTests := []struct {
		name     string
		ret      nvml.Return
		expected bool
	}{
		{
			name:     "String contains 'not supported' (lowercase)",
			ret:      nvml.Return(1000),
			expected: true,
		},
		{
			name:     "String contains 'NOT SUPPORTED' (uppercase)",
			ret:      nvml.Return(1001),
			expected: true,
		},
		{
			name:     "String contains 'Not Supported' (mixed case)",
			ret:      nvml.Return(1002),
			expected: true,
		},
		{
			name:     "String contains 'not supported' with leading/trailing spaces",
			ret:      nvml.Return(1003),
			expected: true,
		},
		{
			name:     "String contains 'not supported' within a longer message",
			ret:      nvml.Return(1004),
			expected: true,
		},
		{
			name:     "String does not contain 'not supported'",
			ret:      nvml.Return(1005),
			expected: false,
		},
		{
			name:     "Empty string",
			ret:      nvml.Return(1006),
			expected: false,
		},
		{
			name:     "String with similar but not exact match",
			ret:      nvml.Return(1007),
			expected: false,
		},
	}

	tests = append(tests, stringBasedTests...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotSupportError(tt.ret)
			assert.Equal(t, tt.expected, result, "IsNotSupportError(%v) = %v, want %v", tt.ret, result, tt.expected)
		})
	}
}

// TestIsNotSupportErrorStringMatch tests the string-based matching of not supported errors
func TestIsNotSupportErrorStringMatch(t *testing.T) {
	// Create a custom Return type that will produce different error strings
	tests := []struct {
		name     string
		ret      nvml.Return
		expected bool
	}{
		{
			name:     "String contains 'not supported' (lowercase)",
			ret:      nvml.Return(1000), // This will produce "Unknown Error" which we'll handle in ErrorString
			expected: true,
		},
		{
			name:     "String contains 'NOT SUPPORTED' (uppercase)",
			ret:      nvml.Return(1001),
			expected: true,
		},
		{
			name:     "String contains 'Not Supported' (mixed case)",
			ret:      nvml.Return(1002),
			expected: true,
		},
		{
			name:     "String contains 'not supported' with leading/trailing spaces",
			ret:      nvml.Return(1003),
			expected: true,
		},
		{
			name:     "String contains 'not supported' within a longer message",
			ret:      nvml.Return(1004),
			expected: true,
		},
		{
			name:     "String does not contain 'not supported'",
			ret:      nvml.Return(1005),
			expected: false,
		},
		{
			name:     "Empty string",
			ret:      nvml.Return(1006),
			expected: false,
		},
		{
			name:     "String with similar but not exact match",
			ret:      nvml.Return(1007),
			expected: false,
		},
	}

	// Override nvml.ErrorString for testing
	originalErrorString := nvml.ErrorString
	defer func() {
		nvml.ErrorString = originalErrorString
	}()

	nvml.ErrorString = func(ret nvml.Return) string {
		switch ret {
		case 1000:
			return "operation is not supported on this device"
		case 1001:
			return "THIS OPERATION IS NOT SUPPORTED"
		case 1002:
			return "Feature Not Supported"
		case 1003:
			return "  not supported  "
		case 1004:
			return "The requested operation is not supported on device 0"
		case 1005:
			return "Some other error"
		case 1006:
			return ""
		case 1007:
			return "notsupported" // No space between 'not' and 'supported'
		default:
			return originalErrorString(ret)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotSupportError(tt.ret)
			assert.Equal(t, tt.expected, result, "IsNotSupportError(%v) = %v, want %v", tt.ret, result, tt.expected)
		})
	}
}

func TestIsVersionMismatchError(t *testing.T) {
	tests := []struct {
		name     string
		ret      nvml.Return
		expected bool
	}{
		{
			name:     "Direct ERROR_ARGUMENT_VERSION_MISMATCH match",
			ret:      nvml.ERROR_ARGUMENT_VERSION_MISMATCH,
			expected: true,
		},
		{
			name:     "Success is not a version mismatch error",
			ret:      nvml.SUCCESS,
			expected: false,
		},
		{
			name:     "Unknown error is not a version mismatch error",
			ret:      nvml.ERROR_UNKNOWN,
			expected: false,
		},
		{
			name:     "Not supported error is not a version mismatch error",
			ret:      nvml.ERROR_NOT_SUPPORTED,
			expected: false,
		},
	}

	// Override nvml.ErrorString for testing string-based matches
	originalErrorString := nvml.ErrorString
	defer func() {
		nvml.ErrorString = originalErrorString
	}()

	nvml.ErrorString = func(ret nvml.Return) string {
		if ret == nvml.Return(1000) {
			return "operation failed due to version mismatch"
		}
		if ret == nvml.Return(1001) {
			return "ERROR: VERSION MISMATCH DETECTED"
		}
		if ret == nvml.Return(1002) {
			return "The API call failed: Version Mismatch between components"
		}
		return originalErrorString(ret)
	}

	// Add string-based test cases
	stringBasedTests := []struct {
		name     string
		ret      nvml.Return
		expected bool
	}{
		{
			name:     "String contains 'version mismatch' (lowercase)",
			ret:      nvml.Return(1000),
			expected: true,
		},
		{
			name:     "String contains 'VERSION MISMATCH' (uppercase)",
			ret:      nvml.Return(1001),
			expected: true,
		},
		{
			name:     "String contains 'Version Mismatch' within message",
			ret:      nvml.Return(1002),
			expected: true,
		},
	}

	tests = append(tests, stringBasedTests...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsVersionMismatchError(tt.ret)
			assert.Equal(t, tt.expected, result, "IsVersionMismatchError(%v) = %v, want %v", tt.ret, result, tt.expected)
		})
	}
}

func TestIsNotReadyError(t *testing.T) {
	tests := []struct {
		name     string
		ret      nvml.Return
		expected bool
	}{
		{
			name:     "Direct ERROR_NOT_READY match",
			ret:      nvml.ERROR_NOT_READY,
			expected: true,
		},
		{
			name:     "Success is not a not-ready error",
			ret:      nvml.SUCCESS,
			expected: false,
		},
		{
			name:     "Unknown error is not a not-ready error",
			ret:      nvml.ERROR_UNKNOWN,
			expected: false,
		},
		{
			name:     "Not supported error is not a not-ready error",
			ret:      nvml.ERROR_NOT_SUPPORTED,
			expected: false,
		},
	}

	// Override nvml.ErrorString for testing string-based matches
	originalErrorString := nvml.ErrorString
	defer func() {
		nvml.ErrorString = originalErrorString
	}()

	nvml.ErrorString = func(ret nvml.Return) string {
		switch ret {
		case nvml.Return(1000):
			return "System is not in ready state"
		case nvml.Return(1001):
			return "SYSTEM IS NOT IN READY STATE"
		case nvml.Return(1002):
			return "nvml.CLOCK_GRAPHICS: System is not in ready state"
		case nvml.Return(1003):
			return "  not in ready  "
		case nvml.Return(1004):
			return "The system is not in ready state for this operation"
		case nvml.Return(1005):
			return "Some other error"
		case nvml.Return(1006):
			return ""
		case nvml.Return(1007):
			return "notinready" // No space between words
		default:
			return originalErrorString(ret)
		}
	}

	// Add string-based test cases
	stringBasedTests := []struct {
		name     string
		ret      nvml.Return
		expected bool
	}{
		{
			name:     "String contains 'not in ready' (lowercase)",
			ret:      nvml.Return(1000),
			expected: true,
		},
		{
			name:     "String contains 'NOT IN READY' (uppercase)",
			ret:      nvml.Return(1001),
			expected: true,
		},
		{
			name:     "String contains 'not in ready' with prefix",
			ret:      nvml.Return(1002),
			expected: true,
		},
		{
			name:     "String contains 'not in ready' with spaces",
			ret:      nvml.Return(1003),
			expected: true,
		},
		{
			name:     "String contains 'not in ready' within message",
			ret:      nvml.Return(1004),
			expected: true,
		},
		{
			name:     "String does not contain 'not in ready'",
			ret:      nvml.Return(1005),
			expected: false,
		},
		{
			name:     "Empty string",
			ret:      nvml.Return(1006),
			expected: false,
		},
		{
			name:     "String with similar but not exact match",
			ret:      nvml.Return(1007),
			expected: false,
		},
	}

	tests = append(tests, stringBasedTests...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotReadyError(tt.ret)
			assert.Equal(t, tt.expected, result, "IsNotReadyError(%v) = %v, want %v", tt.ret, result, tt.expected)
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		ret      nvml.Return
		expected bool
	}{
		{
			name:     "Direct ERROR_NOT_FOUND match",
			ret:      nvml.ERROR_NOT_FOUND,
			expected: true,
		},
		{
			name:     "Success is not a not-found error",
			ret:      nvml.SUCCESS,
			expected: false,
		},
		{
			name:     "Unknown error is not a not-found error",
			ret:      nvml.ERROR_UNKNOWN,
			expected: false,
		},
		{
			name:     "Not supported error is not a not-found error",
			ret:      nvml.ERROR_NOT_SUPPORTED,
			expected: false,
		},
	}

	// Override nvml.ErrorString for testing string-based matches
	originalErrorString := nvml.ErrorString
	defer func() {
		nvml.ErrorString = originalErrorString
	}()

	nvml.ErrorString = func(ret nvml.Return) string {
		switch ret {
		case nvml.Return(1000):
			return "process not found"
		case nvml.Return(1001):
			return "PROCESS NOT FOUND"
		case nvml.Return(1002):
			return "Device Not Found"
		case nvml.Return(1003):
			return "  not found  "
		case nvml.Return(1004):
			return "The requested object was not found on device 0"
		case nvml.Return(1005):
			return "Object not_found in database"
		case nvml.Return(1006):
			return "Some other error"
		case nvml.Return(1007):
			return ""
		case nvml.Return(1008):
			return "notfound" // No space between 'not' and 'found'
		default:
			return originalErrorString(ret)
		}
	}

	// Add string-based test cases
	stringBasedTests := []struct {
		name     string
		ret      nvml.Return
		expected bool
	}{
		{
			name:     "String contains 'not found' (lowercase)",
			ret:      nvml.Return(1000),
			expected: true,
		},
		{
			name:     "String contains 'NOT FOUND' (uppercase)",
			ret:      nvml.Return(1001),
			expected: true,
		},
		{
			name:     "String contains 'Not Found' (mixed case)",
			ret:      nvml.Return(1002),
			expected: true,
		},
		{
			name:     "String contains 'not found' with leading/trailing spaces",
			ret:      nvml.Return(1003),
			expected: true,
		},
		{
			name:     "String contains 'not found' within a longer message",
			ret:      nvml.Return(1004),
			expected: true,
		},
		{
			name:     "String contains 'not_found'",
			ret:      nvml.Return(1005),
			expected: true,
		},
		{
			name:     "String does not contain 'not found' or 'not_found'",
			ret:      nvml.Return(1006),
			expected: false,
		},
		{
			name:     "Empty string",
			ret:      nvml.Return(1007),
			expected: false,
		},
		{
			name:     "String with similar but not exact match",
			ret:      nvml.Return(1008),
			expected: false,
		},
	}

	tests = append(tests, stringBasedTests...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.ret)
			assert.Equal(t, tt.expected, result, "IsNotFoundError(%v) = %v, want %v", tt.ret, result, tt.expected)
		})
	}
}

func TestIsGPULostError(t *testing.T) {
	// Create a function to patch nvml.ErrorString for testing
	originalErrorString := nvml.ErrorString
	defer func() {
		nvml.ErrorString = originalErrorString
	}()

	// Test cases
	testCases := []struct {
		name     string
		ret      nvml.Return
		message  string
		expected bool
	}{
		{
			name:     "ERROR_GPU_IS_LOST constant",
			ret:      nvml.ERROR_GPU_IS_LOST,
			message:  "unused in this case",
			expected: true,
		},
		{
			name:     "message contains 'gpu lost'",
			ret:      nvml.Return(9999), // custom error code
			message:  "the gpu lost error occurred",
			expected: true,
		},
		{
			name:     "message contains 'gpu is lost'",
			ret:      nvml.Return(9998), // custom error code
			message:  "the gpu is lost error message",
			expected: true,
		},
		{
			name:     "message contains 'gpu_is_lost'",
			ret:      nvml.Return(9997), // custom error code
			message:  "gpu_is_lost encountered",
			expected: true,
		},
		{
			name:     "unrelated error",
			ret:      nvml.ERROR_UNKNOWN,
			message:  "this is an unrelated error",
			expected: false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock nvml.ErrorString function to return our test message
			nvml.ErrorString = func(r nvml.Return) string {
				if r == nvml.ERROR_GPU_IS_LOST {
					return "GPU is lost"
				}
				return tc.message
			}

			// Call the function and verify the result
			result := IsGPULostError(tc.ret)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsNoSuchFileOrDirectoryError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "not found error",
			err:      errors.New("file not found"),
			expected: true,
		},
		{
			name:     "no such file or directory error",
			err:      errors.New("no such file or directory"),
			expected: true,
		},
		{
			name:     "mixed case error",
			err:      errors.New("No SuCh FiLe Or DiReCtoRy"),
			expected: true,
		},
		{
			name:     "different error",
			err:      errors.New("permission denied"),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsNoSuchFileOrDirectoryError(test.err)
			if result != test.expected {
				t.Errorf("Expected IsNoSuchFileOrDirectoryError to return %v for error '%v', got %v",
					test.expected, test.err, result)
			}
		})
	}
}
