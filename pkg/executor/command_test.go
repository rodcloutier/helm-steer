package executor

import (
	"fmt"
	"testing"
)

func TestStringOutput(t *testing.T) {

	// --- conditions----------------------------------------------------------
	cmd := NewExecutableCommand("helm", []string{"install", "--name", "foo", "foo/bar"})

	expected := "helm install --name foo foo/bar"

	// --- call ---------------------------------------------------------------
	result := fmt.Sprintf("%s", cmd)

	// --- test ---------------------------------------------------------------
	if result != expected {
		t.Errorf("expected `%s`, got `%s`", expected, result)
	}
}
