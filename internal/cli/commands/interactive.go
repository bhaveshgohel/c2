package commands

import (
	"log"

	"github.com/abiosoft/ishell"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"gitlab.com/teserakt/c2/internal/cli"
)

var (
	// BlacklistedCobraCommands defines a list of cobra command names to be filtered out when running in interactive mode
	BlacklistedCobraCommands = []string{"c2cli", "help", "completion", "interactive"}
)

type interactiveCommand struct {
	cobraCmd *cobra.Command
	rootCmd  cli.Command
	version  string
}

type ishellCmdAdder interface {
	AddCmd(*ishell.Cmd)
}

var _ cli.Command = &interactiveCommand{}

// NewInteractiveCommand returns a command enabling interactive mode
func NewInteractiveCommand(rootCommand cli.Command, version string) cli.Command {
	interactiveCmd := &interactiveCommand{
		rootCmd: rootCommand,
		version: version,
	}

	cobraCmd := &cobra.Command{
		Use:   "interactive",
		Short: "Enter interactive REPL mode",
		RunE:  interactiveCmd.run,
	}

	interactiveCmd.cobraCmd = cobraCmd

	return interactiveCmd
}

// CobraCmd returns the cobra command
func (c *interactiveCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *interactiveCommand) run(cmd *cobra.Command, args []string) error {
	shell := ishell.New()
	//shell.Println(c.version)
	shell.Println("type 'help' for usage information\n")
	shell.SetPrompt("c2cli➩  ")
	shell.AutoHelp(true)

	c.addCobraCommands(shell, c.rootCmd.CobraCmd())

	shell.Run()

	log.Println("bye!")
	return nil
}

func (c *interactiveCommand) addCobraCommands(ishellCmd ishellCmdAdder, cobraCmd *cobra.Command) {
	subIshellCmd := &ishell.Cmd{
		Name:     cobraCmd.Name(),
		Help:     cobraCmd.Short,
		LongHelp: cobraCmd.Long,
	}

	// Skip blacklisted cobra commands to be added on ishell
	var isBlacklisted bool
	for _, blacklistedCmd := range BlacklistedCobraCommands {
		if blacklistedCmd == cobraCmd.Name() {
			isBlacklisted = true
			break
		}
	}

	if isBlacklisted {
		for _, subCobraCmd := range cobraCmd.Commands() {
			c.addCobraCommands(ishellCmd, subCobraCmd)
		}
	} else {
		ishellCmd.AddCmd(subIshellCmd)

		if !cobraCmd.HasSubCommands() {
			subIshellCmd.Func = func(ctx *ishell.Context) {
				// Execute the root cobra command exactly as we would
				// have when running the standard cli.
				cobraRoot := c.rootCmd.CobraCmd()
				cobraRoot.SetArgs(ctx.RawArgs)
				err := cobraRoot.Execute()
				if err != nil {
					ctx.Err(err)
				}

				// As we keep working with the same cobraCmd instance
				// we must restore the flags back to their default value
				// otherwise their last get persisted between calls
				cobraCmd.Flags().Visit(func(f *pflag.Flag) {
					f.Value.Set(f.DefValue)
				})
			}
		} else {
			for _, subCobraCmd := range cobraCmd.Commands() {
				c.addCobraCommands(subIshellCmd, subCobraCmd)
			}
		}
	}
}
