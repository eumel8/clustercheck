package gatecheck

import (
	"os"
	"strings"
	"testing"
)

func TestCheckResult(t *testing.T) {
	// Test CheckResult structure
	result := CheckResult{
		Name:    "Test Check",
		Passed:  true,
		Message: "All systems operational",
	}

	if result.Name != "Test Check" {
		t.Errorf("Expected name 'Test Check', got '%s'", result.Name)
	}

	if !result.Passed {
		t.Error("Expected Passed to be true")
	}

	if result.Message != "All systems operational" {
		t.Errorf("Expected message 'All systems operational', got '%s'", result.Message)
	}
}

func TestGateCheckResult(t *testing.T) {
	// Test GateCheckResult structure
	result := GateCheckResult{
		TotalChecks:  10,
		PassedChecks: 8,
		FailedChecks: 2,
		HealthScore:  80.0,
		CheckResults: []CheckResult{
			{Name: "Check 1", Passed: true, Message: "OK"},
			{Name: "Check 2", Passed: false, Message: "Failed"},
		},
		OverallPassed: true,
	}

	if result.TotalChecks != 10 {
		t.Errorf("Expected TotalChecks 10, got %d", result.TotalChecks)
	}

	if result.PassedChecks != 8 {
		t.Errorf("Expected PassedChecks 8, got %d", result.PassedChecks)
	}

	if result.FailedChecks != 2 {
		t.Errorf("Expected FailedChecks 2, got %d", result.FailedChecks)
	}

	if result.HealthScore != 80.0 {
		t.Errorf("Expected HealthScore 80.0, got %.1f", result.HealthScore)
	}

	if !result.OverallPassed {
		t.Error("Expected OverallPassed to be true")
	}

	if len(result.CheckResults) != 2 {
		t.Errorf("Expected 2 check results, got %d", len(result.CheckResults))
	}
}

func TestGateCheckWithInvalidConfig(t *testing.T) {
	// Save original env vars
	originalKubeConfig := os.Getenv("KUBECONFIG")
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("KUBECONFIG", originalKubeConfig)
		os.Setenv("HOME", originalHome)
	}()

	// Set invalid kubeconfig path
	os.Setenv("KUBECONFIG", "/nonexistent/path/to/kubeconfig")

	_, err := GateCheck("", false, "", false)
	if err == nil {
		t.Error("Expected error for invalid kubeconfig, got nil")
	}

	// GateCheck should fail but not panic
	if err != nil && !strings.Contains(err.Error(), "failed") {
		t.Logf("Got expected error: %v", err)
	}
}
