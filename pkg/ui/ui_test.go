package ui

import (
	"context"
	"testing"
	"time"
)

func TestBold(t *testing.T) {
	result := Bold("hello %s", "world")
	if result == "" {
		t.Error("Bold returned empty string")
	}
}

func TestDim(t *testing.T) {
	result := Dim("hello %s", "world")
	if result == "" {
		t.Error("Dim returned empty string")
	}
}

func TestGreen(t *testing.T) {
	result := Green("[x]")
	if result == "" {
		t.Error("Green returned empty string")
	}
}

func TestCyan(t *testing.T) {
	result := Cyan("[y]")
	if result == "" {
		t.Error("Cyan returned empty string")
	}
}

func TestMagenta(t *testing.T) {
	result := Magenta("file")
	if result == "" {
		t.Error("Magenta returned empty string")
	}
}

func TestWhite(t *testing.T) {
	result := White("text")
	if result == "" {
		t.Error("White returned empty string")
	}
}

func TestSpinnerText(t *testing.T) {
	result := SpinnerText("loading %s", "data")
	if result == "" {
		t.Error("SpinnerText returned empty string")
	}
}

func TestSelectedFile(t *testing.T) {
	result := SelectedFile("main.go", "modified")
	if result == "" {
		t.Error("SelectedFile returned empty string")
	}
}

func TestUnselectedFile(t *testing.T) {
	result := UnselectedFile("main.go", "modified")
	if result == "" {
		t.Error("UnselectedFile returned empty string")
	}
}

func TestSpinner_StartStop(t *testing.T) {
	s := StartSpinner("testing...")
	time.Sleep(250 * time.Millisecond) // let it spin a few frames
	s.Stop()
}

func TestSpinner_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_ = StartSpinnerWithContext(ctx, "testing...")
	time.Sleep(250 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond) // let goroutine exit
}

func TestPrompt(t *testing.T) {
	// Should not panic
	Prompt([]PromptOption{
		{Key: "y", Label: "commit"},
		{Key: "n", Label: "cancel"},
	})
}

func TestPrintMessage(t *testing.T) {
	// Should not panic
	PrintMessage("Test label:", "feat: test message")
}

func TestOutputFunctions(t *testing.T) {
	// These write to stdout/stderr — just verify they don't panic
	Success("ok %s", "done")
	Warn("warning %s", "test")
	Error("error %s", "test")
	Info("info %s", "test")
}
