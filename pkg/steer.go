package steer

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
)

type ChartSpec struct {
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

type Plan struct {
	Charts []ChartSpec `json:"charts"`
}

var skippedInstallFlags = []string{
	"chart",
	"name",
	"name-template",
	"replace",
}

func Steer(planPath string) error {

	// TODO make sure helm is in the path
	// TODO make sure helm is initialized

	// Read the plan.yaml file specified
	content, err := ioutil.ReadFile(planPath)
	if err != nil {
		return err
	}

	var plan Plan
	err = yaml.Unmarshal(content, &plan)
	if err != nil {
		fmt.Println("err:%v\n", err)
		return err
	}

	// For each chart, run the install command
	// if not present
	//      install
	for _, c := range plan.Charts {
		c.install()
	}
	// else
	//      upgrade

	return nil
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
	args = append([]string{"install"}, args...)
	args = append(args, c.Chart)

	fmt.Printf("helm %s\n", strings.Trim(fmt.Sprint(args), "[]"))

	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}
