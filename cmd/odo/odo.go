package main

import (
	"fmt"
	"os"

	"github.com/cdrage/complete"
	"github.com/golang/glog"
	"github.com/openshift/odo/pkg/odo/cli"
	"github.com/openshift/odo/pkg/odo/cli/version"
	"github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/openshift/odo/pkg/preference"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func main() {
	root := cli.NewCmdOdo(cli.OdoRecommendedName, cli.OdoRecommendedName)
	rootCmp := createCompletion(root)
	cmp := complete.New("odo", rootCmp)

	var completionCmd = &cobra.Command{
		Use:   "completion",
		Short: "foo",
		Long:  "foo",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("hi!")
		},
	}
	completionCmd.Annotations = map[string]string{"command": "utility"}
	completionCmd.SetUsageTemplate(util.CmdUsageTemplate)

	cmp.CLI.InstallName = "complete"
	cmp.CLI.UninstallName = "uncomplete"
	cmp.AddFlags(completionCmd.Flags())

	root.AddCommand(completionCmd)

	// add the completion flags to the root command, though they won't appear in completions
	//root.Flags().AddGoFlagSet(pflag.CommandLine)

	// override usage so that flag.Parse uses root command's usage instead of default one when invoked with -h
	pflag.Usage = func() {
		_ = root.Help()
	}

	// parse the flags but hack around to avoid exiting with error code 2 on help
	pflag.CommandLine.Init(os.Args[0], pflag.ContinueOnError)
	args := os.Args[1:]
	if err := pflag.CommandLine.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			os.Exit(0)
		}
	}

	// run the completion, in case that the completion was invoked
	// and ran as a completion script or handled a flag that passed
	// as argument, the Run method will return true,
	// in that case, our program have nothing to do and should return.
	if cmp.Complete() {
		return
	}

	// Call commands
	// checking the value of updatenotification in config
	// before proceeding with fetching the latest version
	cfg, err := preference.New()
	if err != nil {
		util.LogErrorAndExit(err, "")
	}
	if cfg.GetUpdateNotification() {
		updateInfo := make(chan string)
		go version.GetLatestReleaseInfo(updateInfo)

		util.LogErrorAndExit(root.Execute(), "")
		select {
		case message := <-updateInfo:
			fmt.Println(message)
		default:
			glog.V(4).Info("Could not get the latest release information in time. Never mind, exiting gracefully :)")
		}
	} else {
		util.LogErrorAndExit(root.Execute(), "")
	}

}

func createCompletion(root *cobra.Command) complete.Command {
	rootCmp := complete.Command{}
	rootCmp.Flags = make(complete.Flags)
	addFlags := func(pflag *pflag.Flag) {
		if pflag.Hidden {
			return
		}
		var handler complete.Predictor
		handler, ok := completion.GetCommandFlagHandler(root, pflag.Name)
		if !ok {
			handler = complete.PredictAnything
		}

		if len(pflag.Shorthand) > 0 {
			rootCmp.Flags["-"+pflag.Shorthand] = handler
		}

		rootCmp.Flags["--"+pflag.Name] = handler
	}
	root.LocalFlags().VisitAll(addFlags)
	root.InheritedFlags().VisitAll(addFlags)
	if root.HasAvailableSubCommands() {
		rootCmp.Sub = make(complete.Commands)
		for _, c := range root.Commands() {
			if !c.Hidden {
				rootCmp.Sub[c.Name()] = createCompletion(c)
			}
		}
	}

	var handler complete.Predictor
	handler, ok := completion.GetCommandHandler(root)
	if !ok {
		handler = complete.PredictNothing
	}
	rootCmp.Args = handler

	return rootCmp
}
