package command

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli"

	apiv1 "github.com/leptonai/gpud/api/v1"
	client "github.com/leptonai/gpud/client/v1"
	"github.com/leptonai/gpud/pkg/config"
	gpudstate "github.com/leptonai/gpud/pkg/gpud-state"
	"github.com/leptonai/gpud/pkg/login"
	"github.com/leptonai/gpud/pkg/server"
	"github.com/leptonai/gpud/pkg/sqlite"
)

func cmdLogin(cliContext *cli.Context) error {
	token := cliContext.String("token")
	if token == "" {
		fmt.Print("Please visit https://dashboard.lepton.ai/ under Settings/Tokens to fetch your token\nPlease enter your token:")
		if _, err := fmt.Scanln(&token); err != nil && err.Error() != "unexpected newline" {
			return fmt.Errorf("failed reading input: %w", err)
		}
	}
	if token == "" {
		return ErrEmptyToken
	}

	rootCtx, rootCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer rootCancel()

	if err := client.BlockUntilServerReady(
		rootCtx,
		fmt.Sprintf("https://localhost:%d", config.DefaultGPUdPort),
	); err != nil {
		return err
	}

	stateFile, err := config.DefaultStateFile()
	if err != nil {
		return fmt.Errorf("failed to get state file: %w", err)
	}
	dbRO, err := sqlite.Open(stateFile, sqlite.WithReadOnly(true))
	if err != nil {
		return fmt.Errorf("failed to open state file: %w", err)
	}
	defer dbRO.Close()

	machineID, err := gpudstate.ReadMachineID(rootCtx, dbRO)
	if err != nil {
		return err
	}
	if machineID != "" {
		fmt.Printf("machine ID %s already assigned (skipping login)\n", machineID)
		return nil
	}

	endpoint := cliContext.String("endpoint")

	// machine ID has not been assigned yet
	// thus request one and blocks until the login request is processed
	loginResp, err := login.SendRequest(rootCtx, endpoint, apiv1.LoginRequest{Token: token})
	if err != nil {
		return err
	}
	machineID = loginResp.MachineID

	// consume the login response to persist the machine ID
	dbRW, err := sqlite.Open(stateFile)
	if err != nil {
		return fmt.Errorf("failed to open state file: %w", err)
	}
	defer dbRW.Close()
	if err := gpudstate.RecordMachineID(rootCtx, dbRW, dbRO, machineID); err != nil {
		return fmt.Errorf("failed to record machine ID: %w", err)
	}

	fifoFile, err := config.DefaultFifoFile()
	if err != nil {
		return fmt.Errorf("failed to get fifo file: %w", err)
	}

	if err := server.WriteToken(token, fifoFile); err != nil {
		return fmt.Errorf("failed to write token: %v", err)
	}

	if err = gpudstate.UpdateLoginInfo(rootCtx, dbRW, machineID, token); err != nil {
		fmt.Println("machine logged in but failed to update token:", err)
	}

	fmt.Printf("%s successfully logged in with machine id %s\n", checkMark, loginResp.MachineID)
	return nil
}
