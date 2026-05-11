package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/sentiolabs/open-events/internal/codegen"
	"github.com/sentiolabs/open-events/internal/registry"
	"github.com/spf13/cobra"
)

var errGenerationFailed = errors.New("generation failed")

func newGenerateCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "generate <go|python> <registry-path> <output-dir>",
		Short: "Generate code from an OpenEvents registry",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			reg, loadDiags := registry.Load(args[1])
			if loadDiags.HasErrors() {
				fmt.Fprintln(errOut, loadDiags.Error())
				return errGenerationFailed
			}
			validationDiags := registry.Validate(reg)
			if validationDiags.HasErrors() {
				fmt.Fprintln(errOut, validationDiags.Error())
				return errGenerationFailed
			}

			var err error
			switch target {
			case "go":
				err = codegen.GenerateGo(reg, args[2])
			case "python":
				err = codegen.GeneratePython(reg, args[2])
			default:
				return fmt.Errorf("unsupported generation target %q", target)
			}
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}
			fmt.Fprintf(out, "ok: generated %s code in %s\n", target, args[2])
			return nil
		},
	}
}
