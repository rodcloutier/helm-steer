package steer

import (
	"reflect"
	"sort"

	"github.com/rodcloutier/helm-steer/pkg/helm"
)

type ChartSpec struct {
	// matches(^a-zA-Z0-9$\-\.)
	Chart       string   `valid:"required" json:"chart"`
	Devel       string   `valid:"optional" json:"devel"`
	DryRun      bool     `valid:"optional" json:"dry-run"`
	Keyring     string   `valid:"optional" json:"keyring"`
	Namespace   string   `valid:"optional" json:"namespace"`
	NoHooks     bool     `valid:"optional" json:"no-hooks"`
	Set         []string `valid:"optional" json:"set"`
	Timeout     int      `valid:"optional" json:"timeout"`
	TLS         bool     `valid:"optional" json:"tls"`
	TLS_CA_cert string   `valid:"optional" json:"tls-ca-cert"`
	TLS_cert    string   `valid:"optional" json:"tls-cert"`
	TLS_key     string   `valid:"optional" json:"tls-key"`
	TLS_verify  bool     `valid:"optional" json:"tls-verify"`
	Values      []string `valid:"optional" json:"values"`
	Verify      bool     `valid:"optional" json:"verify"`
	Version     string   `valid:"semver,optional" json:"version"`
	Wait        bool     `valid:"optional" json:"wait"`

	// Install specific
	Name         string `valid:"optional" json:"name"`
	NameTemplate string `valid:"optional" json:"name-template"`
	Replace      bool   `valid:"optional" json:"replace"`

	// Upgrade specific
	RecreatePods bool `valid:"optional" json:"recreate-pods"`
	ResetValues  bool `valid:"optional" json:"reset-values"`
	ReuseValues  bool `valid:"optional" json:"reuse-values"`
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

func (c *ChartSpec) install() error {
	skippedFlags := []string{
		"chart",
		"recreate-pods",
		"reset-values",
		"reuse-values",
	}

	args := c.buildHelmCmdArgs(skippedFlags)
	args = append(args, c.Chart)

	return helm.Run("install", args, dryRun)
}

func (c *ChartSpec) upgrade() error {
	skippedFlags := []string{
		"chart",
		"name",
		"name-template",
		"replace",
	}

	args := c.buildHelmCmdArgs(skippedFlags)
	args = append(args, c.Name, c.Chart)

	return helm.Run("upgrade", args, dryRun)
}
