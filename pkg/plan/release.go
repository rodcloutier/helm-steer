package plan

import (
	"fmt"
	"reflect"
	"strconv"
)

type InstallFlags struct {
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

type UpgradeFlags struct {
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

type DeleteFlags struct {
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

type RollbackFlags struct {
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

type ReleaseOperationsFlags struct {
	Install  InstallFlags  `json:"install"`
	Upgrade  UpgradeFlags  `json:"upgrade"`
	Delete   DeleteFlags   `json:"delete"`
	Rollback RollbackFlags `json:"rollback"`
}

type ReleaseSpec struct {
	name      string
	namespace string

	// Exported to json values
	Chart string                 `json:"chart"`
	Flags ReleaseOperationsFlags `json:"flags"`
}

func buildHelmCmdFlags(i interface{}) []string {
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

func (r *ReleaseSpec) Conform(namespace, name string) error {

	r.name = name
	r.namespace = namespace

	r.Flags.Install.Name = name
	r.Flags.Install.Namespace = namespace

	r.Flags.Upgrade.Namespace = namespace

	return nil
}

func (r ReleaseSpec) Version() string {
	return r.Flags.Install.Version
}

// String returns the string representation of a ReleaseSpec
func (r ReleaseSpec) String() string {

	chart := r.Chart
	if r.Flags.Install.Version != "" {
		chart = fmt.Sprintf("%s-%s", chart, r.Flags.Install.Version)
	}

	return fmt.Sprintf("%s chart: %s namespace: %s", r.name, chart, r.namespace)
}

func (r *ReleaseSpec) installCmd() []string {
	args := []string{"install"}
	args = append(args, buildHelmCmdFlags(r.Flags.Install)...)
	return append(args, r.Chart)
}

func (r *ReleaseSpec) upgradeCmd() []string {
	args := []string{"upgrade"}
	args = append(args, buildHelmCmdFlags(r.Flags.Upgrade)...)
	return append(args, r.name, r.Chart)
}

func (r *ReleaseSpec) rollbackCmd(revision int32) []string {
	args := []string{"rollback"}
	args = append(args, buildHelmCmdFlags(r.Flags.Rollback)...)
	return append(args, r.name, strconv.Itoa(int(revision)))
}

func (r *ReleaseSpec) deleteCmd() []string {
	args := []string{"delete"}
	args = append(args, buildHelmCmdFlags(r.Flags.Delete)...)
	return append(args, r.name)
}
