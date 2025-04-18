// Package query implements various NVIDIA-related system queries.
// All interactions with NVIDIA data sources are implemented under the query packages.
package query

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/leptonai/gpud/pkg/log"
	"github.com/leptonai/gpud/pkg/nvidia-query/nvml"
	"github.com/leptonai/gpud/pkg/nvidia-query/peermem"
)

// Get all nvidia component queries.
func Get(ctx context.Context) (o *Output, err error) {
	o = &Output{
		Time: time.Now().UTC(),
	}

	log.Logger.Debugw("counting gpu devices")
	o.GPUDeviceCount, err = CountAllDevicesFromDevDir()
	if err != nil {
		log.Logger.Warnw("failed to count gpu devices", "error", err)
	}

	log.Logger.Debugw("checking lsmod peermem")
	cctx, ccancel := context.WithTimeout(ctx, 30*time.Second)
	o.LsmodPeermem, err = peermem.CheckLsmodPeermemModule(cctx)
	ccancel()
	if err != nil {
		// ignore "context.DeadlineExceeded" since it's not a critical error and it's non-actionable
		if !errors.Is(err, context.DeadlineExceeded) {
			o.LsmodPeermemErrors = append(o.LsmodPeermemErrors, err.Error())
		} else {
			log.Logger.Warnw("lsmod peermem check timed out", "error", err)
		}
	}

	instance, err := nvml.NewInstance(ctx)
	if err != nil {
		return nil, err
	}

	// TODO
	// this may timeout when the GPU is broken
	// e.g.,
	// "nvAssertOkFailedNoLog: Assertion failed: Call timed out [NV_ERR_TIMEOUT]"
	o.NVML, err = instance.Get()
	if err != nil {
		log.Logger.Warnw("nvml get failed", "error", err)
		o.NVMLErrors = append(o.NVMLErrors, err.Error())
	}

	productName := o.GPUProductName()
	if productName != "" {
		o.MemoryErrorManagementCapabilities = SupportedMemoryMgmtCapsByGPUProduct(o.GPUProductName())
	} else {
		log.Logger.Warnw("no gpu product name found -- skipping evaluating memory error management capabilities")
	}
	o.MemoryErrorManagementCapabilities.Message = fmt.Sprintf("GPU product name: %q", productName)

	return o, nil
}

const (
	StateKeyGPUProductName      = "gpu_product_name"
	StateKeyFabricManagerExists = "fabric_manager_exists"
	StateKeyIbstatExists        = "ibstat_exists"
)

type Output struct {
	// Time is the time when the query is executed.
	Time time.Time `json:"time"`

	// GPU device count from the /dev directory.
	GPUDeviceCount int `json:"gpu_device_count"`

	LsmodPeermem       *peermem.LsmodPeermemModuleOutput `json:"lsmod_peermem,omitempty"`
	LsmodPeermemErrors []string                          `json:"lsmod_peermem_errors,omitempty"`

	NVML       *nvml.Output `json:"nvml,omitempty"`
	NVMLErrors []string     `json:"nvml_errors,omitempty"`

	MemoryErrorManagementCapabilities MemoryErrorManagementCapabilities `json:"memory_error_management_capabilities,omitempty"`
}

func (o *Output) YAML() ([]byte, error) {
	return yaml.Marshal(o)
}

func (o *Output) GPUCount() int {
	if o == nil {
		return 0
	}

	cnts := 0
	if o.NVML != nil && len(o.NVML.DeviceInfos) > 0 {
		cnts = len(o.NVML.DeviceInfos)
	}

	return cnts
}

func (o *Output) GPUCountFromNVML() int {
	if o == nil {
		return 0
	}
	if o.NVML == nil {
		return 0
	}
	return len(o.NVML.DeviceInfos)
}

func (o *Output) GPUProductName() string {
	if o == nil {
		return ""
	}

	if o.NVML != nil && len(o.NVML.DeviceInfos) > 0 && o.NVML.DeviceInfos[0].Name != "" {
		return o.NVML.DeviceInfos[0].Name
	}

	return ""
}

// This is the same product name in nvidia-smi outputs.
// ref. https://developer.nvidia.com/management-library-nvml
func (o *Output) GPUProductNameFromNVML() string {
	if o == nil {
		return ""
	}
	if o.NVML != nil && len(o.NVML.DeviceInfos) > 0 {
		return o.NVML.DeviceInfos[0].Name
	}
	return ""
}

const (
	inProgress  = "\033[33m⌛\033[0m"
	checkMark   = "\033[32m✔\033[0m"
	wrongSign   = "\033[31m✘\033[0m"
	warningSign = "\033[33m⚠️\033[0m"
)

func (o *Output) PrintInfo(opts ...OpOption) {
	options := &Op{}
	if err := options.applyOpts(opts); err != nil {
		log.Logger.Warnw("failed to apply options", "error", err)
	}

	fmt.Printf("%s GPU device count '%d' (from /dev)\n", checkMark, o.GPUDeviceCount)
	fmt.Printf("%s GPU count '%d'\n", checkMark, o.GPUCountFromNVML())
	fmt.Printf("%s GPU product name '%s'\n", checkMark, o.GPUProductNameFromNVML())

	if len(o.LsmodPeermemErrors) > 0 {
		fmt.Printf("%s lsmod peermem check failed with %d error(s)\n", wrongSign, len(o.LsmodPeermemErrors))
		for _, err := range o.LsmodPeermemErrors {
			fmt.Println(err)
		}
	} else {
		fmt.Printf("%s successfully checked lsmod peermem\n", checkMark)
	}

	if len(o.NVMLErrors) > 0 {
		fmt.Printf("%s Check failed with %d error(s)\n", wrongSign, len(o.NVMLErrors))
		for _, err := range o.NVMLErrors {
			fmt.Println(err)
		}
	}

	if o.NVML != nil {
		fmt.Printf("%s driver version: %s\n", checkMark, o.NVML.DriverVersion)
		fmt.Printf("%s CUDA version: %s\n", checkMark, o.NVML.CUDAVersion)

		if len(o.NVML.DeviceInfos) > 0 {
			fmt.Printf("%s name: %s\n", checkMark, o.NVML.DeviceInfos[0].Name)
		}

		for _, dev := range o.NVML.DeviceInfos {
			fmt.Printf("\n\n##################\n %s\n\n", dev.UUID)

			if dev.GSPFirmwareMode.Enabled {
				fmt.Printf("%s GSP firmware mode is enabled (supported: %v)\n", checkMark, dev.GSPFirmwareMode.Supported)
			} else {
				fmt.Printf("%s GSP firmware mode is disabled (supported: %v)\n", warningSign, dev.GSPFirmwareMode.Supported)
			}

			// ref. https://docs.nvidia.com/deploy/driver-persistence/index.html
			if dev.PersistenceMode.Enabled {
				fmt.Printf("%s Persistence mode is enabled\n", checkMark)
			} else {
				fmt.Printf("%s Persistence mode is disabled\n", wrongSign)
			}

			if dev.ClockEvents != nil {
				if dev.ClockEvents.HWSlowdown || dev.ClockEvents.HWSlowdownThermal || dev.ClockEvents.HWSlowdownPowerBrake {
					fmt.Printf("%s Found hw slowdown error(s)\n", wrongSign)
					yb, err := dev.ClockEvents.YAML()
					if err != nil {
						log.Logger.Warnw("failed to marshal clock events", "error", err)
					} else {
						fmt.Printf("clock events:\n%s\n\n", string(yb))
					}
				} else {
					fmt.Printf("%s Found no hw slowdown error\n", checkMark)
				}
			}

			if dev.RemappedRows.Supported {
				fmt.Printf("%s Remapped rows supported\n", checkMark)
				if dev.RemappedRows.RequiresReset() {
					fmt.Printf("%s Found that the GPU needs a reset\n", wrongSign)
				}
				if dev.RemappedRows.QualifiesForRMA() {
					fmt.Printf("%s Found that the GPU qualifies for RMA\n", wrongSign)
				}
			} else {
				fmt.Printf("%s Remapped rows are not supported\n", wrongSign)
			}

			uncorrectedErrs := dev.ECCErrors.Volatile.FindUncorrectedErrs()
			if len(uncorrectedErrs) > 0 {
				fmt.Printf("%s found %d ecc volatile uncorrected error(s)\n", wrongSign, len(uncorrectedErrs))
				yb, err := dev.ECCErrors.YAML()
				if err != nil {
					log.Logger.Warnw("failed to marshal ecc errors", "error", err)
				} else {
					fmt.Printf("ecc errors:\n%s\n\n", string(yb))
				}
			} else {
				fmt.Printf("%s Found no ecc volatile uncorrected error\n", checkMark)
			}

			if len(dev.Processes.RunningProcesses) > 0 {
				fmt.Printf("%s Found %d running process\n", checkMark, len(dev.Processes.RunningProcesses))
				yb, err := dev.Processes.YAML()
				if err != nil {
					log.Logger.Warnw("failed to marshal processes", "error", err)
				} else {
					fmt.Printf("\n%s\n\n", string(yb))
				}
			} else {
				fmt.Printf("%s Found no running process\n", checkMark)
			}
		}
	}

	if options.debug {
		copied := *o
		yb, err := copied.YAML()
		if err != nil {
			log.Logger.Warnw("failed to marshal output", "error", err)
		} else {
			fmt.Printf("\n\n##################\nfull nvidia query output\n\n")
			fmt.Println(string(yb))
		}
	}
}
