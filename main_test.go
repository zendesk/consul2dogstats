package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const (
	magicEnvVar = "_C2D_TEST"
)

// Ensure the program exits when no Datadog API key exists.
func TestNoDatadogAPIKey(t *testing.T) {
	ensureProcessExit(t, "TestNoDatadogAPIKey",
		false, "DATADOG_API_KEY environment variable must be set")
}

// Ensure the program exits when Datadog API key exists and is invalid
func TestInvalidDatadogAPIKey(t *testing.T) {
	ensureProcessExit(t, "TestNoDatadogAPIKey",
		false, "Invalid Datadog API key",
		"DATADOG_API_KEY=consul2dogstats_bogus_key")
}

// Ensure the program exits when the collect interval is unparseable
func TestInvalidCollectInterval(t *testing.T) {
	ensureProcessExit(t, "TestNoDatadogAPIKey",
		false, "invalid duration",
		"C2D_COLLECT_INTERVAL=some_bogus_value")
}

func ensureProcessExit(t *testing.T,
	testFunction string, exitSuccess bool, match string, env ...string) {
	if os.Getenv(magicEnvVar) == "1" {
		main()
		return
	}

	output := new(bytes.Buffer)
	env = append(env, magicEnvVar+"=1")

	cmd := exec.Command(os.Args[0], "-test.run="+testFunction)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = output
	cmd.Stderr = output

	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && e.Success() == exitSuccess {
		output := string(output.Bytes())
		if match != "" && !strings.Contains(output, match) {
			t.Log("Output: \n" + output)
			t.Fatalf("output of process did not include %s", match)
		}
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
