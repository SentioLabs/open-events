package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/sentiolabs/open-events/internal/protogen"
	"github.com/sentiolabs/open-events/internal/schemair"
	"github.com/spf13/cobra"
)

var errGenerationFailed = errors.New("generation failed")

func newGenerateCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "generate proto <registry-path> <output-dir>",
		Short: "Generate code from an OpenEvents registry",
		Long:  "Generate protobuf+Buf output from an OpenEvents registry. Downstream language code is produced by protoc plugins (e.g. protoc-gen-go, protoc-gen-python) via Buf.",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			if target != "proto" {
				fmt.Fprintf(errOut, "unsupported generation target %q\n", target)
				return errGenerationFailed
			}

			reg, _, err := loadValidatedRegistry(args[1])
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}

			lock, err := readLockFile(lockFilePath(args[1]))
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}
			if err := schemair.CheckLock(lock, reg); err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}
			ir, err := schemair.FromRegistry(reg, lock)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}
			if err := protogen.Render(ir, args[2]); err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}
			fmt.Fprintf(out, "ok: generated proto schema in %s\n", args[2])
			return nil
		},
	}
}
