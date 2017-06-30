// Copyright © 2017 Rodrigue Cloutier <rodcloutier@gmail.com>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rodcloutier/helm-steer/pkg"
	"github.com/rodcloutier/helm-steer/pkg/format"
)

var (
	// The config file to use for persistent settings
	cfgFile string
	// The namespaces targeted (empty is all namespaces)
	namespaces []string
	// Do not perform the actual options
	dryRun bool
	// The debug flag
	debug bool
	// The verbose flag
	verbose bool
	// The debug writer
	debugWriter io.Writer = ioutil.Discard
	// The output writer
	outputWriter io.Writer = ioutil.Discard
	// The version flag to output the version
	version bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "helm steer [PLAN]",
	Short: "Install multiple charts according to a plan",
	Long:  ``,

	RunE: func(cmd *cobra.Command, args []string) error {

		if version {
			fmt.Println(steer.Version)
			return nil
		}

		if len(args) == 0 {
			// error
			return errors.New("Missing required argument plan file")
		}
		if len(args) > 1 {
			// warning, only the first one will be evaluated
			fmt.Println("warning: Specifiying multiple plans is not currently supported. Only the first one will be processed")
		}

		if debug {
			debugWriter = format.ColorizeWriter(cmd.OutOrStderr(), format.Cyan)
		}
		if verbose {
			outputWriter = cmd.OutOrStderr()
		}

		cmd.SilenceUsage = true

		// TODO move the command execution in a function here to use a closure on the
		// writers?
		return steer.Steer(outputWriter, debugWriter, args[0], namespaces, dryRun)
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Print the executed commands to stderr")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print the executed commands output to stderr")
	RootCmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "only print the operations but does not perform them")
	RootCmd.Flags().StringSliceVarP(&namespaces, "namespace", "n", []string{}, "specify the namespace(s) to target")
	RootCmd.Flags().BoolVarP(&version, "version", "", false, "show the version and exits")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".helm-steer" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".helm-steer")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
