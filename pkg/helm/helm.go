package helm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

// NAME            	REVISION	UPDATED                 	STATUS  	CHART             	NAMESPACE
// alert-marmot    	5       	Sat Jun  3 22:28:08 2017	DEPLOYED	alert-marmot-0.1.0	default
// clunky-clownfish	1       	Sun May 21 08:26:43 2017	DEPLOYED	test-0.1.0        	default
// draft           	1       	Wed Jun  7 20:50:24 2017	DEPLOYED	draftd-canary     	kube-system

// type HelmRelease struct {
// 	Name
// 	Revision int
// 	Updated
// 	Status
// 	Chart
// 	Namespace
// }

func newClient() helm.Interface {
	// options := []helm.Option{helm.Host(settings.TillerHost)}
	options := []helm.Option{helm.Host(os.Getenv("TILLER_HOST"))}
	// if tlsVerify || tlsEnable {
	// 	tlsopts := tlsutil.Options{KeyFile: tlsKeyFile, CertFile: tlsCertFile, InsecureSkipVerify: true}
	// 	if tlsVerify {
	// 		tlsopts.CaCertFile = tlsCaCertFile
	// 		tlsopts.InsecureSkipVerify = false
	// 	}
	// 	tlscfg, err := tlsutil.ClientConfig(tlsopts)
	// 	if err != nil {
	// 		fmt.Fprintln(os.Stderr, err)
	// 		os.Exit(2)
	// 	}
	// 	options = append(options, helm.WithTLS(tlscfg))
	// }
	return helm.NewClient(options...)
}

func Run(name string, args []string, dryRun bool) error {

	args = append([]string{name}, args...)

	fmt.Printf("helm %s\n", strings.Trim(fmt.Sprint(args), "[]"))
	if dryRun {
		return nil
	}

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

func List() ([]*release.Release, error) {
	client := newClient()

	res, err := client.ListReleases()
	if err != nil {
		return []*release.Release{}, err
	}

	for _, r := range res.Releases {

		// Maybe do this only if r.Config.Values is empty
		var config map[string]interface{}
		err = yaml.Unmarshal([]byte(r.Config.Raw), &config)

		// fmt.Println(config)
	}

	return res.Releases, nil
}