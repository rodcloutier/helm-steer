package plan

import (
	"testing"
)

func TestPlanValidity(t *testing.T) {

	// Single namespace is by default valid
	p := Plan{
		Version: "beta1",
		Namespaces: map[string]Namespace{
			"foo": Namespace{
				Releases: map[string]Release{
					"service": Release{},
				},
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
			"foo": Namespace{
				Releases: map[string]Release{
					"service": Release{},
				},
			},
			"bar": Namespace{
				Releases: map[string]Release{
					"service": Release{},
				},
			},
		},
	}

	valid, _ = p.verify()
	if valid {
		t.Error("Expected to have a duplicate but none found")
	}
}
