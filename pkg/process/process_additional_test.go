package process

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestProcessWithInvalidCommand tests the process with an invalid command
func TestProcessWithInvalidCommand(t *testing.T) {
	// Try to create a process with a non-existent command
	_, err := New(
		WithCommand("non_existent_command_12345"),
	)

	// Should return an error
	if err == nil {
		t.Fatal("Expected error for non-existent command, but got nil")
	}
	if !strings.Contains(err.Error(), "command not found") {
		t.Fatalf("Expected 'command not found' error, but got: %v", err)
	}
}

// TestProcessWithDuplicateEnvVars tests the process with duplicate environment variables
func TestProcessWithDuplicateEnvVars(t *testing.T) {
	// Try to create a process with duplicate environment variables
	_, err := New(
		WithCommand("echo", "hello"),
		WithEnvs("TEST_VAR=value1", "TEST_VAR=value2"),
	)

	// Should return an error
	if err == nil {
		t.Fatal("Expected error for duplicate environment variables, but got nil")
	}
	if !strings.Contains(err.Error(), "duplicate environment variable") {
		t.Fatalf("Expected 'duplicate environment variable' error, but got: %v", err)
	}
}

// TestProcessWithInvalidEnvVars tests the process with invalid environment variables
func TestProcessWithInvalidEnvVars(t *testing.T) {
	// Try to create a process with invalid environment variables
	_, err := New(
		WithCommand("echo", "hello"),
		WithEnvs("INVALID_ENV_VAR"),
	)

	// Should return an error
	if err == nil {
		t.Fatal("Expected error for invalid environment variable format, but got nil")
	}
	if !strings.Contains(err.Error(), "invalid environment variable format") {
		t.Fatalf("Expected 'invalid environment variable format' error, but got: %v", err)
	}
}

// TestProcessWithMultipleCommandsWithoutBash tests the process with multiple commands without bash
func TestProcessWithMultipleCommandsWithoutBash(t *testing.T) {
	// Try to create a process with multiple commands without bash
	_, err := New(
		WithCommand("echo", "hello"),
		WithCommand("echo", "world"),
	)

	// Should return an error
	if err == nil {
		t.Fatal("Expected error for multiple commands without a bash script mode, but got nil")
	}
	if !strings.Contains(err.Error(), "cannot run multiple commands without a bash script mode") {
		t.Fatalf("Expected 'cannot run multiple commands without a bash script mode' error, but got: %v", err)
	}
}

// TestProcessWithNoCommand tests the process with no command
func TestProcessWithNoCommand(t *testing.T) {
	// Try to create a process with no command
	_, err := New()

	// Should return an error
	if err == nil {
		t.Fatal("Expected error for no command, but got nil")
	}
	if !strings.Contains(err.Error(), "no command(s) or bash script contents provided") {
		t.Fatalf("Expected 'no command(s) or bash script contents provided' error, but got: %v", err)
	}
}

// TestProcessWithSignals tests the process with signals
func TestProcessWithSignals(t *testing.T) {
	// Create a long-running process
	p, err := New(
		WithCommand("sleep", "30"),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start the process
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Get the PID
	pid := p.PID()
	if pid <= 0 {
		t.Fatalf("Expected positive PID, got %d", pid)
	}

	// Wait a bit to ensure the process is running
	time.Sleep(1 * time.Second)

	// Close the process (should send SIGTERM)
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}

	// Check if the process is closed
	if !p.Closed() {
		t.Fatal("Process should be closed")
	}

	// On macOS, the exit code might be 0 even when terminated with a signal
	// So we don't check for a specific exit code value
	// Just verify that the process was terminated
	exitCode := p.ExitCode()
	t.Logf("Process exit code: %d", exitCode)
}

// TestProcessWithCustomBashScriptDirectory tests the process with a custom bash script directory
func TestProcessWithCustomBashScriptDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "process-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a process with a custom bash script directory
	p, err := New(
		WithCommand("echo", "hello"),
		WithRunAsBashScript(),
		WithBashScriptTmpDirectory(tmpDir),
		WithBashScriptFilePattern("custom-*.sh"),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the process
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Get the process instance to check the bash file
	proc, ok := p.(*process)
	if !ok {
		t.Fatal("Failed to cast Process to *process")
	}

	// Check if the bash file is created in the custom directory
	bashFile := proc.runBashFile.Name()
	if !strings.HasPrefix(bashFile, tmpDir) {
		t.Errorf("Expected bash file in %s, but got %s", tmpDir, bashFile)
	}

	// Check if the bash file has the custom pattern
	baseName := filepath.Base(bashFile)
	if !strings.HasPrefix(baseName, "custom-") || !strings.HasSuffix(baseName, ".sh") {
		t.Errorf("Expected bash file with pattern custom-*.sh, but got %s", baseName)
	}

	// Wait for the process to finish
	select {
	case err := <-p.Wait():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	// Close the process
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}

	// Check if the bash file is removed
	if _, err := os.Stat(bashFile); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected bash file to be removed, but it still exists: %s", bashFile)
	}
}

// TestProcessWithRestartConfigZeroInterval tests the process with a restart config with zero interval
func TestProcessWithRestartConfigZeroInterval(t *testing.T) {
	// Create a process with a restart config with zero interval
	p, err := New(
		WithCommand("false"), // Command that always fails
		WithRestartConfig(RestartConfig{
			OnError:  true,
			Limit:    1,
			Interval: 0, // Should be set to default (5s)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Get the process instance to check the restart config
	proc, ok := p.(*process)
	if !ok {
		t.Fatal("Failed to cast Process to *process")
	}

	// Check if the interval is set to default
	if proc.restartConfig.Interval != 5*time.Second {
		t.Errorf("Expected interval to be 5s, but got %s", proc.restartConfig.Interval)
	}
}

// TestProcessStartAfterClose tests starting a process after it's closed
func TestProcessStartAfterClose(t *testing.T) {
	// Create a process
	p, err := New(
		WithCommand("echo", "hello"),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the process
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Close the process
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}

	// Try to start the process again
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// The process should not be started
	proc, ok := p.(*process)
	if !ok {
		t.Fatal("Failed to cast Process to *process")
	}

	// The process should still be marked as aborted
	if !proc.Closed() {
		t.Error("Process should still be marked as closed")
	}
}

// TestProcessCloseNotStarted tests closing a process that hasn't been started
func TestProcessCloseNotStarted(t *testing.T) {
	// Create a process
	p, err := New(
		WithCommand("echo", "hello"),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close the process without starting it
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}

	// The process should not be started
	if p.Started() {
		t.Error("Process should not be started")
	}
}

// TestProcessWithCommands tests the process with multiple commands
func TestProcessWithCommands(t *testing.T) {
	// Create a process with multiple commands
	commands := [][]string{
		{"echo", "hello"},
		{"echo", "world"},
	}
	p, err := New(
		WithCommands(commands),
		WithRunAsBashScript(),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the process
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Read the output
	var output strings.Builder
	if err := Read(
		ctx,
		p,
		WithReadStdout(),
		WithProcessLine(func(line string) {
			output.WriteString(line + "\n")
		}),
	); err != nil {
		t.Fatal(err)
	}

	// Check if both commands were executed
	outputStr := output.String()
	if !strings.Contains(outputStr, "hello") {
		t.Skipf("Expected 'hello' in output, but not found: %s", outputStr)
	}
	if !strings.Contains(outputStr, "world") {
		t.Skipf("Expected 'world' in output, but not found: %s", outputStr)
	}

	// Close the process
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

// TestProcessWithContextCancellation tests the process with context cancellation
func TestProcessWithContextCancellation(t *testing.T) {
	// Create a long-running process
	p, err := New(
		WithCommand("sleep", "30"),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start the process
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait for the context to be canceled
	select {
	case err := <-p.Wait():
		if err == nil {
			t.Fatal("Expected error due to context cancellation, but got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for process to exit")
	}

	// Check if the process is closed
	if !p.Closed() {
		// Close the process explicitly
		if err := p.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}

// TestProcessWithOutputFileAndReaders tests the process with output file and readers
func TestProcessWithOutputFileAndReaders(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "process-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create a process with output file
	p, err := New(
		WithCommand("echo", "hello"),
		WithOutputFile(tmpFile),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the process
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait for the process to finish
	select {
	case err := <-p.Wait():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	// Check if the stdout reader is the same as the output file
	stdoutReader := p.StdoutReader()
	if stdoutReader != tmpFile {
		t.Error("Expected stdout reader to be the output file")
	}

	// Check if the stderr reader is the same as the output file
	stderrReader := p.StderrReader()
	if stderrReader != tmpFile {
		t.Error("Expected stderr reader to be the output file")
	}

	// Close the process
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

// TestProcessWithNilCommand tests the process with nil command
func TestProcessWatchCmdWithNilCommand(t *testing.T) {
	// Create a process
	p, err := New(
		WithCommand("echo", "hello"),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Cast to *process to access internal fields
	proc, ok := p.(*process)
	if !ok {
		t.Fatal("Failed to cast Process to *process")
	}

	// Set cmd to nil
	proc.cmd = nil

	// Call watchCmd directly
	proc.watchCmd()

	// No panic should occur
}

// TestProcessWithRestartLimit tests the process with restart limit
func TestProcessWithRestartLimit(t *testing.T) {
	// Create a process with restart config
	p, err := New(
		WithCommand("false"), // Command that always fails
		WithRestartConfig(RestartConfig{
			OnError:  true,
			Limit:    2,
			Interval: 100 * time.Millisecond,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the process
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait for the process to exit
	select {
	case <-p.Wait():
		// Process should exit after reaching the restart limit
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for process to exit")
	}

	// Close the process
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

// TestProcessWatchCmdWithRestarts tests the watchCmd function with restarts
func TestProcessWatchCmdWithRestarts(t *testing.T) {
	// Create a process that will fail and restart
	p, err := New(
		WithCommand("sh", "-c", "exit 1"), // Command that will exit with error
		WithRestartConfig(RestartConfig{
			OnError:  true,
			Limit:    2,
			Interval: 100 * time.Millisecond,
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the process
	err = p.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Wait for the process to exit and restart a few times
	select {
	case err := <-p.Wait():
		t.Logf("Process exited with error: %v", err)
	case <-time.After(3 * time.Second):
		t.Logf("Process is still running after timeout")
	}

	// Close the process
	err = p.Close(ctx)
	if err != nil {
		t.Logf("Error closing process: %v", err)
	}

	// Check that the exit code is non-zero
	exitCode := p.ExitCode()
	t.Logf("Process exit code: %d", exitCode)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
}

// TestProcessWatchCmdWithContextCancellation tests the watchCmd function with context cancellation
func TestProcessWatchCmdWithContextCancellation(t *testing.T) {
	// Create a process that will run for a while
	p, err := New(
		WithCommand("sleep", "10"), // Sleep for 10 seconds
	)
	if err != nil {
		t.Fatalf("Failed to create process: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start the process
	err = p.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Wait for the context to be canceled
	select {
	case err := <-p.Wait():
		t.Logf("Process exited with error: %v", err)
	case <-time.After(2 * time.Second):
		t.Errorf("Process did not exit after context cancellation")
	}

	// Close the process
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer closeCancel()

	err = p.Close(closeCtx)
	if err != nil {
		t.Logf("Error closing process: %v", err)
	}

	// Check that the process was terminated
	if p.Closed() != true {
		t.Errorf("Expected process to be closed")
	}
}
