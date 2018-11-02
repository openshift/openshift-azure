// Package api defines the external API for the plugin.
package api

import (
	"fmt"
	"testing"
)

func TestPluginError(t *testing.T) {
	want := "WaitForConsoleHealth: test error string"
	pe := &PluginError{
		Err:  fmt.Errorf("test error string"),
		Step: PluginStepWaitForConsoleHealth,
	}
	if got := pe.Error(); got != want {
		t.Errorf("PluginError.Error() = %v, want %v", got, want)
	}
}
