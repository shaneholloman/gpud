package command

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"

	"github.com/leptonai/gpud/pkg/accelerator"
	"github.com/leptonai/gpud/pkg/asn"
	"github.com/leptonai/gpud/pkg/log"
	"github.com/leptonai/gpud/pkg/netutil"
	latency_edge "github.com/leptonai/gpud/pkg/netutil/latency/edge"
	"github.com/leptonai/gpud/pkg/process"
)

func cmdJoin(cliContext *cli.Context) (retErr error) {
	rootCtx, rootCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer rootCancel()
	endpoint := cliContext.String("endpoint")
	clusterName := cliContext.String("cluster-name")
	provider := cliContext.String("provider")
	nodeGroup := cliContext.String("node-group")
	extraInfo := cliContext.String("extra-info")
	privateIP := cliContext.String("private-ip")

	uid, err := GetUID(rootCtx)
	if err != nil {
		return err
	}

	cmd := exec.Command("nproc", "--all")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("executing nproc: %w", err)
	}

	totalCPU, err := strconv.ParseInt(strings.TrimSpace(out.String()), 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing cpu: %w", err)
	}

	_, productName, err := accelerator.DetectTypeAndProductName()
	if err != nil {
		return err
	}

	// network section
	region := "unknown"
	latencies, _ := latency_edge.Measure(rootCtx)
	if len(latencies) > 0 {
		closest := latencies.Closest()
		region = closest.RegionCode
	}

	detectProvider := "unknown"
	publicIP, _ := netutil.PublicIP()
	asnResult, err := asn.GetASLookup(publicIP)
	if err != nil {
		log.Logger.Errorf("failed to get asn lookup: %v", err)
	} else {
		detectProvider = asnResult.AsnName
	}

	if !cliContext.Bool("skip-interactive") {
		reader := bufio.NewReader(os.Stdin)
		var input string
		if productName != "unknown" {
			fmt.Printf("We detect your gpu type is %v, if this is correct, press Enter. If not, please enter your gpu shape below\n", productName)
			input, err = reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading input:", err)
				return
			}
			if input != "\n" {
				productName = strings.TrimSpace(input)
			}
		}

		fmt.Printf("We detect your public IP is %v, if this is correct, press Enter. If not, please enter your public IP below\n", publicIP)
		input, err = reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		if input != "\n" {
			publicIP = strings.TrimSpace(input)
		}

		if provider == "" {
			fmt.Printf("Provider name not specified, we detected your provider is %v, if correct, press Enter. If not, please enter your provider's name below\n", detectProvider)
			input, err = reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading input:", err)
				return
			}
			if input != "\n" {
				provider = strings.TrimSpace(input)
			} else {
				provider = detectProvider
			}
		}

		fmt.Printf("We detect your region is %v, if this is correct, press Enter. If not, please enter your region below\n", region)
		input, err = reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		if input != "\n" {
			region = strings.TrimSpace(input)
		}
	} else {
		if provider == "" {
			provider = detectProvider
		}
	}

	type payload struct {
		ID               string `json:"id"`
		ClusterName      string `json:"cluster_name"`
		PublicIP         string `json:"public_ip"`
		Provider         string `json:"provider"`
		ProviderGPUShape string `json:"provider_gpu_shape"`
		TotalCPU         int64  `json:"total_cpu"`
		NodeGroup        string `json:"node_group"`
		ExtraInfo        string `json:"extra_info"`
		Region           string `json:"region"`
		PrivateIP        string `json:"private_ip"`
	}
	type RespErr struct {
		Error  string `json:"error"`
		Status string `json:"status"`
	}
	content := payload{
		ID:               uid,
		ClusterName:      clusterName,
		PublicIP:         publicIP,
		Provider:         strings.Replace(provider, " ", "-", -1),
		ProviderGPUShape: productName,
		TotalCPU:         totalCPU,
		NodeGroup:        nodeGroup,
		ExtraInfo:        extraInfo,
		Region:           region,
		PrivateIP:        privateIP,
	}
	rawPayload, _ := json.Marshal(&content)
	fmt.Println("Your machine will be initialized with following configuration, please press Enter if it is ok")
	prettyJSON, _ := json.MarshalIndent(content, "", "  ")
	fmt.Println(string(prettyJSON))
	fmt.Printf("%sWarning: GPUd will upgrade your container runtime to containerd, will affect your current running containers (if any)%s\n", "\033[33m", "\033[0m")
	fmt.Printf("%sWarning: GPUd will Reboot your machine to finish necessary setup%s\n", "\033[33m", "\033[0m")
	fmt.Printf("Please look carefully about the above warning, if ok, please hit Enter\n")
	if !cliContext.Bool("skip-interactive") {
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if input != "\n" {
			fmt.Println("Non empty input received, GPUd join aborted.")
			return nil
		}
	}
	fmt.Println("Please wait while control plane is initializing basic setup for your machine, this may take up to one minute...")
	response, err := http.Post(fmt.Sprintf("%s/api/v1/join", endpoint), "application/json", bytes.NewBuffer(rawPayload))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %w", err)
		}
		var errorResponse RespErr
		err = json.Unmarshal(body, &errorResponse)
		if err != nil {
			return fmt.Errorf("Error parsing error response: %v\nResponse body: %s", err, body)
		}
		return fmt.Errorf("failed to join: %v", errorResponse)
	}
	cctx, ccancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer ccancel()
	if err := handleJoinResponse(cctx, response.Body); err != nil {
		return err
	}
	fmt.Println("Basic setup finished, GPUd is installing necessary components onto your machine, this may take 10 - 15 minutes.\nYou can run `gpud status` or `gpud status -w` to check the progress of each component.")
	return nil
}

func handleJoinResponse(ctx context.Context, body io.Reader) error {
	dir, err := untarFiles("/tmp/", body)
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(dir, "join.sh")
	return runCommand(ctx, scriptPath, nil)
}

func untarFiles(targetDir string, body io.Reader) (string, error) {
	var dir string
	gzipReader, err := gzip.NewReader(body)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		fpath := filepath.Join(targetDir, header.Name)
		if dir == "" {
			dir = fpath
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				panic(err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(fpath)
			if err != nil {
				panic(err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				panic(err)
			}
		}
	}
	return dir, nil
}

func runCommand(ctx context.Context, script string, result *string) error {
	var ops []process.OpOption

	p, err := process.New(append(ops, process.WithCommand("bash", script))...)
	if err != nil {
		return err
	}
	if result != nil {
		go func() {
			stdoutReader := p.StdoutReader()
			if stdoutReader == nil {
				log.Logger.Errorf("failed to read stdout: %v", err)
				return
			}
			rawResult, err := io.ReadAll(p.StdoutReader())
			if err != nil {
				log.Logger.Errorf("failed to read stout: %v", err)
				return
			}
			*result = strings.TrimSpace(string(rawResult))
		}()
	}
	if err = p.Start(ctx); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-p.Wait():
		if err != nil {
			return err
		}
	}
	if err := p.Close(ctx); err != nil {
		return err
	}
	return nil
}
