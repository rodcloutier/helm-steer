package plan

import (
	"testing"
)

func TestPlanValidity(t *testing.T) {

	// Single namespace is by default valid
	p := Plan{
		Version: "beta1",
		Namespaces: map[string]Namespace{
			"foo": map[string]Stack{
				"service": Stack{},
			},
		},
	}

	valid, _ := p.verify()
	if !valid {
		t.Errorf("Expected single namespace to be valid")
	}

	// Duplicate names in namspaces is not valid
	p = Plan{
		Version: "beta1",
		Namespaces: map[string]Namespace{
			"foo": map[string]Stack{
				"service": Stack{},
			},
			"bar": map[string]Stack{
				"service": Stack{},
			},
		},
	}

	valid, _ = p.verify()
	if valid {
		t.Error("Expected to have a duplicate but none found")
	}
}
