package process

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProcess(t *testing.T) {
	p, err := New(
		WithCommand("echo", "hello"),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("pid: %d", p.PID())

	// redunant start is ok
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	if err := Read(
		ctx,
		p,
		WithReadStdout(),
		WithProcessLine(func(line string) {
			t.Logf("stdout: %q", line)
		}),
	); err != nil {
		t.Fatal(err)
	}

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if !p.Closed() {
		t.Fatal("process is not aborted")
	}
}

func TestProcessRunBashScriptContents(t *testing.T) {
	p, err := New(
		WithBashScriptContentsToRun(`#!/bin/bash

# do not mask errors in a pipeline
set -o pipefail

echo "hello"
`),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("pid: %d", p.PID())

	b, err := io.ReadAll(p.StderrReader())
	if err != nil {
		if !strings.Contains(err.Error(), "file already closed") {
			t.Fatal(err)
		}
	}
	t.Logf("stderr: %q", string(b))

	b, err = io.ReadAll(p.StdoutReader())
	if err != nil {
		if !strings.Contains(err.Error(), "file already closed") {
			t.Fatal(err)
		}
	}
	t.Logf("stdout: %q", string(b))

	select {
	case err := <-p.Wait():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	proc, _ := p.(*process)
	if proc.Closed() {
		t.Fatal("process is closed")
	}
	bashFile := proc.runBashFile.Name()
	if bashFile == "" {
		t.Fatal("bash file is not created")
	}

	if _, err := os.Stat(bashFile); err != nil {
		t.Fatal(err)
	}

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	// redunant abort is ok
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}

	if !proc.Closed() {
		t.Fatal("process is not closed")
	}
	if _, err := os.Stat(bashFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}

func TestProcessWithBash(t *testing.T) {
	p, err := New(
		WithCommand("echo", "hello"),
		WithCommand("echo hello && echo 111 | grep 1"),
		WithRunAsBashScript(),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("pid: %d", p.PID())

	select {
	case err := <-p.Wait():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestProcessWithTempFile(t *testing.T) {
	// create a temporary file
	tmpFile, err := os.CreateTemp("", "process-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	p, err := New(
		WithCommand("echo", "hello"),
		WithOutputFile(tmpFile),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("pid: %d", p.PID())

	select {
	case err := <-p.Wait():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}

	// Verify the content of the temporary file
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expectedContent := "hello\n"
	if string(content) != expectedContent {
		t.Fatalf("Expected content %q, but got %q", expectedContent, string(content))
	}
}

func TestProcessWithStdoutReader(t *testing.T) {
	p, err := New(
		WithCommand("echo hello && sleep 1000"),
		WithRunAsBashScript(),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("pid: %d", p.PID())

	select {
	case err := <-p.Wait():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
	}

	rd := p.StdoutReader()
	buf := make([]byte, 1024)
	n, err := rd.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	output := string(buf[:n])
	expectedOutput := "hello\n"
	if output != expectedOutput {
		t.Fatalf("expected output %q, but got %q", expectedOutput, output)
	}
	t.Logf("stdout: %q", output)

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestProcessWithStdoutReaderUntilEOF(t *testing.T) {
	p, err := New(
		WithCommand("echo hello 1 && sleep 1"),
		WithCommand("echo hello 2 && sleep 1"),
		WithCommand("echo hello 3 && sleep 1"),
		WithRunAsBashScript(),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("pid: %d", p.PID())

	rd := p.StdoutReader()
	scanner := bufio.NewScanner(rd)
	var output string
	for scanner.Scan() {
		output += scanner.Text() + "\n"
	}
	expectedOutput := "hello 1\nhello 2\nhello 3\n"
	if output != expectedOutput {
		t.Fatalf("expected output %q, but got %q", expectedOutput, output)
	}
	t.Logf("stdout: %q", output)

	select {
	case err := <-p.Wait():
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
	}

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if scanner.Err() != nil {
		t.Fatal(scanner.Err())
	}
}

func TestProcessWithRestarts(t *testing.T) {
	p, err := New(
		WithCommand("echo hello"),
		WithCommand("echo 111 && exit 1"),
		WithRunAsBashScript(),
		WithRestartConfig(RestartConfig{
			OnError:  true,
			Limit:    3,
			Interval: 100 * time.Millisecond,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("pid: %d", p.PID())

	for i := 0; i < 3; i++ {
		select {
		case err := <-p.Wait():
			if err == nil {
				t.Fatal("expected error")
			}
			if strings.Contains(err.Error(), "exit status 1") {
				t.Log(err)
				continue
			}
			t.Fatal(err)

		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestProcessSleep(t *testing.T) {
	p, err := New(
		WithCommand("sleep", "99999"),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("pid: %d", p.PID())

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-p.Wait():
		if err == nil {
			t.Fatal("expected error")
		}
		t.Log(err)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func TestProcessStream(t *testing.T) {
	opts := []OpOption{
		WithRunAsBashScript(),
	}
	for i := 0; i < 100; i++ {
		opts = append(opts, WithCommand(fmt.Sprintf("echo hello %d && sleep 1", i)))
	}

	p, err := New(opts...)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("pid: %d", p.PID())

	rd := p.StdoutReader()
	buf := make([]byte, 1024)
	for i := 0; i < 3; i++ {
		n, err := rd.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		output := string(buf[:n])
		expectedOutput := fmt.Sprintf("hello %d\n", i)
		if output != expectedOutput {
			t.Fatalf("expected output %q, but got %q", expectedOutput, output)
		}
		t.Logf("stdout: %q", output)
	}

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestProcessExitCode(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedError  bool
		expectedOutput string
		expectedCode   int32
	}{
		{
			name:           "command with non-zero exit",
			args:           []string{"sh", "-c", "exit 42"},
			expectedError:  true,
			expectedOutput: "",
			expectedCode:   42,
		},
		{
			name:           "successful command",
			args:           []string{"echo", "hello"},
			expectedError:  false,
			expectedOutput: "hello\n",
			expectedCode:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(WithCommand(tt.args...))
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := p.Start(ctx); err != nil {
				t.Fatal(err)
			}

			var output string
			if err := Read(
				ctx,
				p,
				WithReadStdout(),
				WithProcessLine(func(line string) {
					output += line + "\n"
				}),
			); err != nil && !tt.expectedError {
				t.Fatal(err)
			}

			select {
			case err := <-p.Wait():
				if tt.expectedError && err == nil {
					t.Error("expected error but got none")
				}
				if !tt.expectedError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("timeout waiting for process to finish")
			}

			if output != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, output)
			}

			if p.ExitCode() != tt.expectedCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedCode, p.ExitCode())
			}

			if err := p.Close(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}
