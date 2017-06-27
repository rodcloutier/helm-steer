package plan

import (
	"fmt"
	"reflect"
)

type InstallArgs struct {
	CAFile       string   `json:"ca-file"`
	Cert_file    string   `json:"cert-file"`
	Devel        string   `json:"devel"`
	Dry_run      bool     `json:"dry-run"`
	Key_file     string   `json:"key-file"`
	Keyring      string   `json:"keyring"`
	Name         string   `json:"name"`
	NameTemplate string   `json:"name-template"`
	Namespace    string   `json:"namespace"`
	No_hooks     bool     `json:"no-hooks"`
	Replace      bool     `json:"replace"`
	Repo         string   `json:"repo"`
	Set          []string `json:"set"`
	Timeout      int      `json:"timeout"`
	TLS          bool     `json:"tls"`
	TLS_CA_cert  string   `json:"tls-ca-cert"`
	TLS_cert     string   `json:"tls-cert"`
	TLS_key      string   `json:"tls-key"`
	TLS_verify   bool     `json:"tls-verify"`
	Values       []string `json:"values"`
	Verify       bool     `json:"verify"`
	Version      string   `json:"version"`
	Wait         bool     `json:"wait"`
}

type UpgradeArgs struct {
	CAFile        string   `json:"ca-file"`
	Cert_file     string   `json:"cert-file"`
	Devel         string   `json:"devel"`
	Dry_run       bool     `json:"dry-run"`
	Force         bool     `json:"force"`
	Install       bool     `json:"install"`
	Key_file      string   `json:"key-file"`
	Keyring       string   `json:"keyring"`
	Namespace     string   `json:"namespace"`
	No_hooks      bool     `json:"no-hooks"`
	Recreate_pods bool     `json:"recreate-pods"`
	Repo          string   `json:"repo"`
	Reset_values  bool     `json:"reset-values"`
	Reuse_values  bool     `json:"reuse-values"`
	Set           []string `json:"set"`
	Timeout       int      `json:"timeout"`
	TLS           bool     `json:"tls"`
	TLS_CA_cert   string   `json:"tls-ca-cert"`
	TLS_cert      string   `json:"tls-cert"`
	TLS_key       string   `json:"tls-key"`
	TLS_verify    bool     `json:"tls-verify"`
	Values        []string `json:"values"`
	Verify        bool     `json:"verify"`
	Version       string   `json:"version"`
	Wait          bool     `json:"wait"`
}

type DeleteArgs struct {
	Dry_run     bool   `json:"dry-run"`
	No_hooks    bool   `json:"no-hooks"`
	Purge       bool   `json:"purge"`
	Timeout     int    `json:"timeout"`
	TLS         bool   `json:"tls"`
	TLS_CA_cert string `json:"tls-ca-cert"`
	TLS_cert    string `json:"tls-cert"`
	TLS_key     string `json:"tls-key"`
	TLS_verify  bool   `json:"tls-verify"`
}

type RollbackArgs struct {
	Dry_run       bool   `json:"dry-run"`
	Force         bool   `json:"force"`
	No_hooks      bool   `json:"no-hooks"`
	Recreate_pods bool   `json:"recreate-pods"`
	Timeout       int    `json:"timeout"`
	TLS           bool   `json:"tls"`
	TLS_CA_cert   string `json:"tls-ca-cert"`
	TLS_cert      string `json:"tls-cert"`
	TLS_key       string `json:"tls-key"`
	TLS_verify    bool   `json:"tls-verify"`
	Wait          bool   `json:"wait"`
}

type ChartSpec struct {
	name      string
	namespace string

	// Exported to json values
	Chart    string       `json:"chart"`
	Install  InstallArgs  `json:"install"`
	Upgrade  UpgradeArgs  `json:"upgrade"`
	Delete   DeleteArgs   `json:"delete"`
	Rollback RollbackArgs `json:"rollback"`
}

func buildHelmCmdArgs(i interface{}) []string {
	var cmd []string

	val := reflect.Indirect(reflect.ValueOf(i))

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		name := typeField.Tag.Get("json")

		name = "--" + name
		valueField := val.Field(i)

		if !valueField.CanInterface() {
			// The value is not exported, skip it
			continue
		}
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

func (c *ChartSpec) Conform(namespace, name string) error {

	c.name = name
	c.namespace = namespace

	c.Install.Name = name
	c.Install.Namespace = namespace

	c.Upgrade.Namespace = namespace

	return nil
}

func (c ChartSpec) Version() string {
	return c.Install.Version
}

// String returns the string representation of a ChartSpec
func (c ChartSpec) String() string {

	chart := c.Chart
	if c.Install.Version != "" {
		chart = fmt.Sprintf("%s-%s", chart, c.Install.Version)
	}

	return fmt.Sprintf("%s chart: %s namespace: %s", c.name, chart, c.namespace)
}

func (c *ChartSpec) installCmd() []string {
	args := []string{"install"}
	args = append(args, buildHelmCmdArgs(c.Install)...)
	return append(args, c.Chart)
}

func (c *ChartSpec) upgradeCmd() []string {
	args := []string{"upgrade"}
	args = append(args, buildHelmCmdArgs(c.Upgrade)...)
	return append(args, c.name, c.Chart)
}

func (c *ChartSpec) rollbackCmd() []string {
	args := []string{"rollback"}
	args = append(args, buildHelmCmdArgs(c.Rollback)...)
	// TODO (rod) fetch the revision that is expected (the last one)
	revision := "1"
	return append(args, c.name, revision)
}

func (c *ChartSpec) deleteCmd() []string {
	args := []string{"delete"}
	args = append(args, buildHelmCmdArgs(c.Delete)...)
	return append(args, c.name)
}
