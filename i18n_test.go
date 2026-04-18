package zcli

import (
	"bytes"
	"strings"
	"testing"
)

func TestLanguageManagerFallbackAndMissing(t *testing.T) {
	manager := NewLanguageManager("zh")
	manager.primary.Service.Operations.Install = ""

	if got := manager.GetText("service.operations.install"); got != "Install Service" {
		t.Fatalf("expected fallback text, got %q", got)
	}

	if got := manager.GetText("service.operations.unknown"); got != "[Missing: service.operations.unknown]" {
		t.Fatalf("expected missing marker, got %q", got)
	}
}

func TestLanguageManagerUnknownPrimaryDefaultsToEnglish(t *testing.T) {
	manager := NewLanguageManager("unknown")

	if got := manager.GetText("service.operations.install"); got != "Install Service" {
		t.Fatalf("expected English default, got %q", got)
	}
}

func TestServiceLocalizerConfigureOutput(t *testing.T) {
	manager := NewLanguageManager("en")
	localizer := NewServiceLocalizer(manager, nil)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	localizer.ConfigureOutput(&out, &errBuf, false, false)
	localizer.LogInfo("demo", "running")
	localizer.LogError("runFailed", nil)

	if got := out.String(); !strings.Contains(got, "demo") {
		t.Fatalf("expected service name in info output, got %q", got)
	}
	if got := errBuf.String(); !strings.Contains(got, "Failed to run service") {
		t.Fatalf("expected localized error output, got %q", got)
	}
}
