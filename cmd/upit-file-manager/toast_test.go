package main

import (
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestToastPayloadHasOnlySilentVisualContent(t *testing.T) {
	var payload struct {
		XMLName    xml.Name
		Activation string     `xml:"activationType,attr"`
		Launch     string     `xml:"launch,attr"`
		Text       []string   `xml:"visual>binding>text"`
		Actions    []struct{} `xml:"actions>action"`
		Audio      struct {
			Silent bool `xml:"silent,attr"`
		} `xml:"audio"`
	}
	notification := terminalNotification{ID: "unique-event", Title: "Upload complete", Body: "Final URL copied to clipboard."}
	if err := xml.Unmarshal([]byte(windowsToastXML(notification)), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.XMLName.Local != "toast" || payload.Activation != "" || payload.Launch != "" || len(payload.Actions) != 0 || !payload.Audio.Silent {
		t.Fatalf("Toast contains unexpected sound or activation payload: %+v", payload)
	}
	if len(payload.Text) != 2 || payload.Text[0] != "Upload complete" || payload.Text[1] != "Final URL copied to clipboard." {
		t.Fatalf("unexpected/private notification content: %v", payload.Text)
	}
}

func TestToastProcessErrorsDistinguishUnavailableFromAPIFailure(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []string{"unavailable", "api-failure"} {
		t.Run(outcome, func(t *testing.T) {
			cmd := exec.Command(executable, "-test.run=^TestToastProcessBoundary$")
			cmd.Env = append(os.Environ(), "UPIT_TEST_TOAST_PROCESS="+outcome)
			err := runWindowsToast(cmd, terminalNotification{ID: "unique-event", Title: "Upload complete", Body: "Final URL copied to clipboard."})
			if outcome == "unavailable" {
				if !errors.Is(err, errors.ErrUnsupported) {
					t.Fatalf("disabled/missing identity not reported as unavailable: %v", err)
				}
			} else {
				var exitError *exec.ExitError
				if !errors.As(err, &exitError) || errors.Is(err, errors.ErrUnsupported) {
					t.Fatalf("API failure lost its process error: %v", err)
				}
			}
		})
	}
}

// Substitute only the native process boundary, with independent OS exit outcomes.
func TestToastProcessBoundary(t *testing.T) {
	switch os.Getenv("UPIT_TEST_TOAST_PROCESS") {
	case "unavailable":
		os.Exit(2)
	case "api-failure":
		os.Exit(1)
	}
}
