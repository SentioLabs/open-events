package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

func Execute() error {
	return NewRootCommand(os.Stdout, os.Stderr).Execute()
}

func NewRootCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "openevents", Short: "OpenEvents event taxonomy compiler", SilenceUsage: true, SilenceErrors: true}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.AddCommand(newValidateCommand(out, errOut))
	cmd.AddCommand(newGenerateCommand(out, errOut))
	return cmd
}
