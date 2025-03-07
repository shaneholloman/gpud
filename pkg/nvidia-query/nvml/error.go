package nvml

import (
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// IsVersionMismatchError returns true if the error indicates a version mismatch.
func IsVersionMismatchError(ret nvml.Return) bool {
	if ret == nvml.ERROR_ARGUMENT_VERSION_MISMATCH {
		return true
	}

	e := normalizeErrorString(nvml.ErrorString(ret))
	return strings.Contains(e, "version mismatch")
}

// Returns true if the error indicates that the operation is not supported.
func IsNotSupportError(ret nvml.Return) bool {
	if ret == nvml.ERROR_NOT_SUPPORTED {
		return true
	}

	e := normalizeErrorString(nvml.ErrorString(ret))
	return strings.Contains(e, "not supported")
}

// normalizeErrorString normalizes an NVML error string by converting it to lowercase and trimming whitespace.
func normalizeErrorString(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}
