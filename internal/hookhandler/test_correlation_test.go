package hookhandler

import (
	"strings"
	"testing"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
)

func TestExtractTestFailures_Go(t *testing.T) {
	t.Parallel()

	output := `--- FAIL: TestLogin (0.01s)
    handler_test.go:45: expected "admin" got "user"
--- FAIL: TestLogout (0.00s)
    handler_test.go:67: session not cleared
FAIL
exit status 1`

	failures := extractTestFailures(output)
	if len(failures) != 2 {
		t.Fatalf("extractTestFailures() = %d failures, want 2", len(failures))
	}
	if failures[0].TestName != "TestLogin" {
		t.Errorf("failures[0].TestName = %q, want TestLogin", failures[0].TestName)
	}
	if failures[1].TestName != "TestLogout" {
		t.Errorf("failures[1].TestName = %q, want TestLogout", failures[1].TestName)
	}
}

func TestExtractTestFailures_Python(t *testing.T) {
	t.Parallel()

	output := `FAILED tests/test_auth.py::test_login - AssertionError: expected admin
FAILED tests/test_auth.py::test_signup - ValueError: invalid email`

	failures := extractTestFailures(output)
	if len(failures) != 2 {
		t.Fatalf("extractTestFailures() = %d failures, want 2", len(failures))
	}
	if failures[0].TestName != "test_login" {
		t.Errorf("failures[0].TestName = %q, want test_login", failures[0].TestName)
	}
}

func TestExtractTestFailures_NoFailure(t *testing.T) {
	t.Parallel()

	output := `ok  	github.com/example/pkg	0.5s
PASS`

	failures := extractTestFailures(output)
	if len(failures) != 0 {
		t.Errorf("extractTestFailures() = %d failures, want 0 for passing tests", len(failures))
	}
}

func TestCorrelateWithRecentEdits(t *testing.T) {
	t.Parallel()

	sessionID := "test-correlate"
	sdb, err := sessiondb.Open(sessionID)
	if err != nil {
		t.Fatalf("sessiondb.Open() = %v", err)
	}
	t.Cleanup(func() { _ = sdb.Destroy() })

	_ = sdb.AddWorkingSetFile("/src/auth/handler.go")
	_ = sdb.AddWorkingSetFile("/src/auth/middleware.go")

	failures := []testFailure{
		{TestName: "TestLogin", ErrorMessage: `expected "admin" got "user"`},
	}

	result := correlateWithRecentEdits(sdb, failures)
	if result == "" {
		t.Fatal("correlateWithRecentEdits() = empty, want correlation")
	}
	if !strings.Contains(result, "TestLogin") {
		t.Errorf("result missing test name, got: %s", result)
	}
	if !strings.Contains(result, "handler.go") {
		t.Errorf("result missing file, got: %s", result)
	}
}

func TestCorrelateWithRecentEdits_NoFiles(t *testing.T) {
	t.Parallel()

	sessionID := "test-correlate-nofiles"
	sdb, err := sessiondb.Open(sessionID)
	if err != nil {
		t.Fatalf("sessiondb.Open() = %v", err)
	}
	t.Cleanup(func() { _ = sdb.Destroy() })

	failures := []testFailure{
		{TestName: "TestLogin"},
	}

	result := correlateWithRecentEdits(sdb, failures)
	if result != "" {
		t.Errorf("correlateWithRecentEdits() with no files = %q, want empty", result)
	}
}

func TestExtractNearbyError(t *testing.T) {
	t.Parallel()

	output := `--- FAIL: TestLogin (0.01s)
    handler_test.go:45: expected "admin" got "user"
--- FAIL: TestLogout`

	errMsg := extractNearbyError(output, "--- FAIL: TestLogin (0.01s)")
	if errMsg == "" {
		t.Fatal("extractNearbyError() = empty")
	}
	if !strings.Contains(errMsg, "expected") {
		t.Errorf("extractNearbyError() = %q, want to contain 'expected'", errMsg)
	}
}
