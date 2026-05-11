package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/sentiolabs/open-events/internal/registry"
	"github.com/spf13/cobra"
)

var errValidationFailed = errors.New("validation failed")

func newValidateCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "validate <path>",
		Short: "Validate an OpenEvents registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, loadDiags := registry.Load(args[0])
			if loadDiags.HasErrors() {
				fmt.Fprintln(errOut, loadDiags.Error())
				return errValidationFailed
			}

			validationDiags := registry.Validate(reg)
			if validationDiags.HasErrors() {
				fmt.Fprintln(errOut, validationDiags.Error())
				return errValidationFailed
			}

			fmt.Fprintf(out, "ok: registry valid (%d events, %d context fields)\n", len(reg.Events), len(reg.Context))
			return nil
		},
	}
}
