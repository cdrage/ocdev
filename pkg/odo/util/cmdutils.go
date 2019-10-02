package util

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/openshift/odo/pkg/component"
	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/machineoutput"
	"github.com/openshift/odo/pkg/occlient"
	urlPkg "github.com/openshift/odo/pkg/url"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LogErrorAndExit prints the cause of the given error and exits the code with an
// exit code of 1.
// If the context is provided, then that is printed, if not, then the cause is
// detected using errors.Cause(err)
// *If* we are using the global json parameter, we instead output the json output
func LogErrorAndExit(err error, context string, a ...interface{}) {

	if err != nil {

		// If it's JSON, we'll output  the error
		if log.IsJSON() {

			// Machine readble error output
			machineOutput := machineoutput.GenericError{
				TypeMeta: metav1.TypeMeta{
					Kind:       machineoutput.Kind,
					APIVersion: machineoutput.APIVersion,
				},
				Message: err.Error(),
			}

			// Output the error
			machineoutput.OutputError(machineOutput)

		} else {
			glog.V(4).Infof("Error:\n%v", err)
			if context == "" {
				log.Error(errors.Cause(err))
			} else {
				log.Errorf(fmt.Sprintf("%s", strings.Title(context)), a...)
			}
		}

		// Always exit 1 anyways
		os.Exit(1)

	}
}

// CheckOutputFlag validates the -o flag
func CheckOutputFlag(outputFlag string) error {
	switch outputFlag {
	case "", "json":
		return nil
	default:
		return fmt.Errorf("Please input valid output format. available format: json")
	}

}

// PrintComponentInfo prints Component Information like path, URL & storage
func PrintComponentInfo(client *occlient.Client, currentComponentName string, componentDesc component.Component, applicationName string) {
	localConfig, err := config.New()
	if err != nil {
		LogErrorAndExit(err, "")
	}

	log.Describef("Component Name: ", currentComponentName)
	log.Describef("Type: ", componentDesc.Spec.Type)

	// Source
	if componentDesc.Spec.Source != "" {
		log.Describef("Source: ", componentDesc.Spec.Source)
	}

	// Env
	if componentDesc.Spec.Env != nil {

		// Retrieve all the environment variables
		var output string
		for _, env := range componentDesc.Spec.Env {
			output += fmt.Sprintf(" · %v=%v\n", env.Name, env.Value)
		}

		// Cut off the last newline and output
		output = output[:len(output)-1]
		log.Describef("Environment Variables:\n", output)
	}

	// Storage
	if len(componentDesc.Spec.Storage) > 0 {

		// Retrieve the storage list
		storages, err := localConfig.StorageList()
		LogErrorAndExit(err, "")

		// Gather the output
		var output string
		for _, store := range storages {
			output += fmt.Sprintf(" · %v of size %v mounted to %v\n", store.Name, store.Size, store.Path)
		}

		// Cut off the last newline and output
		output = output[:len(output)-1]
		log.Describef("Storage:\n", output)
	}

	// URL
	if componentDesc.Spec.URL != nil {

		// Retrieve the URLs
		urls, err := urlPkg.ListPushed(client, currentComponentName, applicationName)
		LogErrorAndExit(err, "")

		// Gather the output
		var output string
		for _, componentURL := range componentDesc.Spec.URL {
			url := urls.Get(componentURL)
			output += fmt.Sprintf(" · %v exposed via %v\n", urlPkg.GetURLString(url.Spec.Protocol, url.Spec.Host), url.Spec.Port)
		}

		// Cut off the last newline and output
		output = output[:len(output)-1]
		log.Describef("URLs:\n", output)

	}

	// Linked components
	if len(componentDesc.Status.LinkedComponents) > 0 {

		// Gather the output
		var output string
		for name, ports := range componentDesc.Status.LinkedComponents {
			if len(ports) > 0 {
				output += fmt.Sprintf(" · %v - Port(s): %v\n", name, strings.Join(ports, ","))
			} else {
				output += fmt.Sprintf(" · %v\n", name)
			}
		}

		// Cut off the last newline and output
		output = output[:len(output)-1]
		log.Describef("Linked Components:\n", output)

	}

	// Linked services
	if len(componentDesc.Status.LinkedServices) > 0 {

		// Gather the output
		var output string
		for _, linkedService := range componentDesc.Status.LinkedServices {
			output += fmt.Sprintf(" · %s\n", linkedService)
		}

		// Cut off the last newline and output
		output = output[:len(output)-1]
		log.Describef("Linked Services:\n", output)

	}

}

// GetFullName generates a command's full name based on its parent's full name and its own name
func GetFullName(parentName, name string) string {
	return parentName + " " + name
}

// VisitCommands visits each command within Cobra.
// Adapted from: https://github.com/cppforlife/knctl/blob/612840d3c9729b1c57b20ca0450acab0d6eceeeb/pkg/knctl/cobrautil/misc.go#L23
func VisitCommands(cmd *cobra.Command, f func(*cobra.Command)) {
	f(cmd)
	for _, child := range cmd.Commands() {
		VisitCommands(child, f)
	}
}

// CapitalizeFlagDescriptions adds capitalizations
func CapitalizeFlagDescriptions(f *pflag.FlagSet) string {
	f.VisitAll(func(f *pflag.Flag) {
		cap := []rune(f.Usage)
		cap[0] = unicode.ToUpper(cap[0])
		f.Usage = string(cap)
	})
	return f.FlagUsages()
}

// CmdUsageTemplate is the main template used for all command line usage
var CmdUsageTemplate = `Usage:{{if .Runnable}}
  {{if .HasAvailableFlags}}{{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath}} [command]{{end}}{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{CapitalizeFlagDescriptions .LocalFlags | trimRightSpace}}{{end}}{{ if .HasAvailableInheritedFlags}}

Global Flags:
{{CapitalizeFlagDescriptions .InheritedFlags | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableSubCommands }}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// ThrowContextError prints a context error if application/project is not found
func ThrowContextError() error {
	return errors.Errorf(`Please specify the application name and project name
Or use the command from inside a directory containing an odo component.`)
}
