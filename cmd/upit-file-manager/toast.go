package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func windowsToastXML(notification terminalNotification) string {
	var payload strings.Builder
	payload.Grow(160 + len(notification.Title) + len(notification.Body))
	payload.WriteString(`<toast><visual><binding template="ToastGeneric"><text>`)
	_ = xml.EscapeText(&payload, []byte(notification.Title))
	payload.WriteString(`</text><text>`)
	_ = xml.EscapeText(&payload, []byte(notification.Body))
	payload.WriteString(`</text></binding></visual><audio silent="true"/></toast>`)
	return payload.String()
}

func runWindowsToast(cmd *exec.Cmd, notification terminalNotification) error {
	cmd.Env = append(cmd.Environ(), "UPIT_TOAST_XML="+windowsToastXML(notification), "UPIT_TOAST_TAG="+notification.ID)
	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 2 {
			return errors.ErrUnsupported
		}
		return fmt.Errorf("Windows Toast delivery failed: %w", err)
	}
	return nil
}
