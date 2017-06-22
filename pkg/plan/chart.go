package plan

import (
	"fmt"
	"reflect"
	"sort"
)

type ChartSpec struct {
	// Common
	Chart       string   `json:"chart"`
	Devel       string   `json:"devel"`
	DryRun      bool     `json:"dry-run"`
	Keyring     string   `json:"keyring"`
	Namespace   string   `json:"namespace"`
	NoHooks     bool     `json:"no-hooks"`
	Set         []string `json:"set"`
	Timeout     int      `json:"timeout"`
	TLS         bool     `json:"tls"`
	TLS_CA_cert string   `json:"tls-ca-cert"`
	TLS_cert    string   `json:"tls-cert"`
	TLS_key     string   `json:"tls-key"`
	TLS_verify  bool     `json:"tls-verify"`
	Values      []string `json:"values"`
	Verify      bool     `json:"verify"`
	Version     string   `json:"version"`
	Wait        bool     `json:"wait"`

	// Install specific
	Name         string `json:"name"`
	NameTemplate string `json:"name-template"`
	Replace      bool   `json:"replace"`

	// Upgrade specific
	RecreatePods bool `json:"recreate-pods"`
	ResetValues  bool `json:"reset-values"`
	ReuseValues  bool `json:"reuse-values"`
}

func (c *ChartSpec) buildHelmCmdArgs(skippedFields []string) []string {
	var cmd []string

	skippedFields = append(skippedFields, "chart")

	sort.Strings(skippedFields)
	isSkipped := func(s string) bool {
		i := sort.Search(len(skippedFields),
			func(i int) bool { return skippedFields[i] >= s })
		return i < len(skippedFields) && skippedFields[i] == s
	}

	val := reflect.ValueOf(c).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		name := typeField.Tag.Get("json")

		if isSkipped(name) {
			continue
		}

		name = "--" + name
		valueField := val.Field(i)
		value := reflect.ValueOf(valueField.Interface())

		switch kind := typeField.Type.Kind(); kind {
		case reflect.Bool:
			if value.Bool() == true {
				cmd = append(cmd, name)
			}
		case reflect.String:
			v := value.String()
			if v != "" {
				cmd = append(cmd, name, v)
			}

		case reflect.Slice:
			for ii := 0; ii < value.Len(); ii++ {
				v := value.Index(ii).String()
				if v != "" {
					cmd = append(cmd, name, v)
				}
			}
		}
	}

	return cmd
}

// String returns the string representation of a ChartSpec
func (c ChartSpec) String() string {
	return fmt.Sprintf("%s chart: %s-%s namespace: %s", c.Name, c.Chart, c.Version, c.Namespace)
}

func (c *ChartSpec) installCmd() []string {
	skippedFlags := []string{
		"chart",
		"recreate-pods",
		"reset-values",
		"reuse-values",
	}

	args := []string{"install"}
	args = append(args, c.buildHelmCmdArgs(skippedFlags)...)
	return append(args, c.Chart)
}

func (c *ChartSpec) upgradeCmd() []string {
	skippedFlags := []string{
		"chart",
		"name",
		"name-template",
		"replace",
	}

	args := []string{"upgrade"}
	args = append(args, c.buildHelmCmdArgs(skippedFlags)...)
	return append(args, c.Name, c.Chart)
}
