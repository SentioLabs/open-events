package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/sentiolabs/open-events/internal/codegen"
	"github.com/sentiolabs/open-events/internal/protogen"
	"github.com/sentiolabs/open-events/internal/schemair"
	"github.com/spf13/cobra"
)

var errGenerationFailed = errors.New("generation failed")

func newGenerateCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "generate <go|python|proto> <registry-path> <output-dir>",
		Short: "Generate code from an OpenEvents registry",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			reg, _, loadErr := loadValidatedRegistry(args[1])
			if loadErr != nil {
				fmt.Fprintln(errOut, loadErr)
				return errGenerationFailed
			}

			var err error
			switch target {
			case "go":
				err = codegen.GenerateGo(reg, args[2])
			case "python":
				err = codegen.GeneratePython(reg, args[2])
			case "proto":
				var lock schemair.Lock
				lock, err = readLockFile(lockFilePath(args[1]))
				if err == nil {
					err = schemair.CheckLock(lock, reg)
				}
				if err == nil {
					var ir schemair.Registry
					ir, err = schemair.FromRegistry(reg, lock)
					if err == nil {
						err = protogen.Render(ir, args[2])
					}
				}
				if err != nil {
					fmt.Fprintln(errOut, err)
					return errGenerationFailed
				}
				fmt.Fprintf(out, "ok: generated proto schema in %s\n", args[2])
				return nil
			default:
				fmt.Fprintf(errOut, "unsupported generation target %q\n", target)
				return errGenerationFailed
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
