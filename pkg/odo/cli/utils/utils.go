package utils

import (
	"fmt"
	"net/url"
	"runtime"

	"github.com/openshift/odo/pkg/config"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/util"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

const RecommendedCommandName = "utils"

// NewCmdUtils implements the utils odo command
func NewCmdUtils(name, fullName string) *cobra.Command {
	terminalCmd := NewCmdTerminal(terminalCommandName, odoutil.GetFullName(fullName, terminalCommandName))
	utilsCmd := &cobra.Command{
		Use:   name,
		Short: "Utilities for terminal commands and modifying Odo configurations",
		Long:  `Utilities for terminal commands and modifying Odo configurations`,
		Example: fmt.Sprintf("%s\n",
			terminalCmd.Example),
	}

	utilsCmd.Annotations = map[string]string{"command": "utility"}
	utilsCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)

	utilsCmd.AddCommand(terminalCmd)
	return utilsCmd
}

// VisitCommands visits each command within Cobra.
// Adapted from: https://github.com/cppforlife/knctl/blob/612840d3c9729b1c57b20ca0450acab0d6eceeeb/pkg/knctl/cobrautil/misc.go#L23
func VisitCommands(cmd *cobra.Command, f func(*cobra.Command)) {
	f(cmd)
	for _, child := range cmd.Commands() {
		VisitCommands(child, f)
	}
}

// LocalConfigInfo ...
type LocalConfigInfo struct {
	LocalConfig *config.LocalConfigInfo
	SourcePath  string
}

// RetrieveLocalConfigInfo ...
func RetrieveLocalConfigInfo(componentContext string) (configInfo LocalConfigInfo, err error) {

	conf, err := config.NewLocalConfigInfo(componentContext)
	if err != nil {
		return LocalConfigInfo{}, errors.Wrap(err, "failed to fetch component config")
	}

	sourcePath, err := correctSourcePath(conf)
	if err != nil {
		return LocalConfigInfo{}, errors.Wrap(err, "unable to validate source path")
	}

	return LocalConfigInfo{
		LocalConfig: conf,
		SourcePath:  sourcePath}, nil
}

// correctSourcePath corrects the current sourcePath with PushOptions depending on
// local or binary configuration
func correctSourcePath(localConfig *config.LocalConfigInfo) (path string, err error) {

	cmpName := localConfig.GetName()
	sourceType := localConfig.GetSourceType()
	sourcePath := localConfig.GetSourceLocation()

	if sourceType == config.BINARY || sourceType == config.LOCAL {
		u, err := url.Parse(sourcePath)
		if err != nil {
			return "", errors.Wrapf(err, "unable to parse source %s from component %s", sourcePath, cmpName)
		}

		if u.Scheme != "" && u.Scheme != "file" {
			return "", fmt.Errorf("Component %s has invalid source path %s", cmpName, u.Scheme)
		}
		return util.ReadFilePath(u, runtime.GOOS), nil
	}
	return sourcePath, nil
}

// ApplyIgnore ...
func ApplyIgnore(ignores *[]string, sourcePath string) (err error) {
	if len(*ignores) == 0 {
		rules, err := util.GetIgnoreRulesFromDirectory(sourcePath)
		if err != nil {
			odoutil.LogErrorAndExit(err, "")
		}
		*ignores = append(*ignores, rules...)
	}
	return nil
}
