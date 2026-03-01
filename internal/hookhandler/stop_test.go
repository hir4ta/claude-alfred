package hookhandler

import (
	"encoding/json"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/sessiondb"
)

func openStopTestDB(t *testing.T) (string, *sessiondb.SessionDB) {
	t.Helper()
	id := "test-stop-" + t.Name()
	sdb, err := sessiondb.Open(id)
	if err != nil {
		t.Fatalf("sessiondb.Open(%q) = %v", id, err)
	}
	t.Cleanup(func() { _ = sdb.Destroy() })
	return id, sdb
}

func makeStopInput(t *testing.T, sessionID, msg string) []byte {
	t.Helper()
	in := stopInput{
		CommonInput:          CommonInput{SessionID: sessionID},
		LastAssistantMessage: msg,
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("json.Marshal() = %v", err)
	}
	return data
}

func TestHandleStop_EmptyMessage(t *testing.T) {
	t.Parallel()
	id, _ := openStopTestDB(t)
	input := makeStopInput(t, id, "")
	output, err := handleStop(input)
	if err != nil {
		t.Fatalf("handleStop() error = %v", err)
	}
	if output != nil {
		t.Errorf("handleStop(empty) = %+v, want nil", output)
	}
}

func TestHandleStop_CleanCompletion(t *testing.T) {
	t.Parallel()
	id, _ := openStopTestDB(t)
	input := makeStopInput(t, id, "All tasks completed successfully. Tests pass and build is clean.")
	output, err := handleStop(input)
	if err != nil {
		t.Fatalf("handleStop() error = %v", err)
	}
	if output != nil {
		t.Errorf("handleStop(clean) = %+v, want nil (no block)", output)
	}
}

func TestHandleStop_SingleTodoNoBlock(t *testing.T) {
	// Not parallel — handleStop calls SetDeliveryContext which writes package globals.
	// Single text signal → soft warning, not block.
	id, _ := openStopTestDB(t)
	input := makeStopInput(t, id, "Implementation complete. TODO: add edge case tests later.")
	output, err := handleStop(input)
	if err != nil {
		t.Fatalf("handleStop() error = %v", err)
	}
	if output != nil && output.Decision == "block" {
		t.Error("handleStop(single TODO) should not block, want soft warning only")
	}
}

func TestHandleStop_UnresolvedFailure(t *testing.T) {
	t.Parallel()
	id, sdb := openStopTestDB(t)
	_ = sdb.RecordFailure("Bash", "test", "build error: undefined func", "main.go")
	input := makeStopInput(t, id, "I've made some changes to main.go.")
	output, err := handleStop(input)
	if err != nil {
		t.Fatalf("handleStop() error = %v", err)
	}
	if output == nil || output.Decision != "block" {
		t.Errorf("handleStop(unresolved failure) should block, got %+v", output)
	}
}

func TestHandleStop_MultipleSignals(t *testing.T) {
	t.Parallel()
	id, _ := openStopTestDB(t)
	input := makeStopInput(t, id, "TODO: fix the remaining test failures. The build is still failing.")
	output, err := handleStop(input)
	if err != nil {
		t.Fatalf("handleStop() error = %v", err)
	}
	if output == nil || output.Decision != "block" {
		t.Errorf("handleStop(multiple signals) should block, got %+v", output)
	}
}

func TestHandleStop_Japanese(t *testing.T) {
	// Japanese incomplete + placeholder → 2 signals → block.
	t.Parallel()
	id, _ := openStopTestDB(t)
	input := makeStopInput(t, id, "実装完了。残りのテストは後で追加します。TODO: エッジケース")
	output, err := handleStop(input)
	if err != nil {
		t.Fatalf("handleStop() error = %v", err)
	}
	// "残り" (incompletePatterns) + "TODO" (placeholderPatterns) = 2 signals → block.
	if output == nil || output.Decision != "block" {
		t.Errorf("handleStop(Japanese+TODO) should block, got %+v", output)
	}
}

func TestHandleStop_JapaneseSingleSignal(t *testing.T) {
	// Not parallel — handleStop calls SetDeliveryContext which writes package globals.
	// Single Japanese signal → soft warning, no block.
	id, _ := openStopTestDB(t)
	input := makeStopInput(t, id, "実装完了。残りのテストは後で追加します。")
	output, err := handleStop(input)
	if err != nil {
		t.Fatalf("handleStop() error = %v", err)
	}
	// "残り" is detected (single signal) → no block.
	if output != nil && output.Decision == "block" {
		t.Error("handleStop(single Japanese signal) should not block")
	}
}
