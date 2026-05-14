package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	golang "github.com/sentiolabs/open-events/internal/codegen/golang"
	python "github.com/sentiolabs/open-events/internal/codegen/python"
	"github.com/sentiolabs/open-events/internal/constgen"
	"github.com/sentiolabs/open-events/internal/protogen"
	"github.com/sentiolabs/open-events/internal/schemair"
	"github.com/spf13/cobra"
)

var errGenerationFailed = errors.New("generation failed")

func newGenerateCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate <registry-path>",
		Short: "Generate language bindings from an OpenEvents registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registryPath := args[0]

			reg, _, err := loadValidatedRegistry(registryPath)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}

			lock, err := readLockFile(lockFilePath(registryPath))
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

			// Emit proto to <registry>/.openevents/proto/
			protoOut := filepath.Join(registryPath, ".openevents", "proto")
			if err := protogen.Render(ir, protoOut); err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}

			// Dispatch to each configured language emitter.
			var emitted []string
			for _, lang := range reg.Codegen.Languages {
				rawCfg := reg.Codegen.Configs[lang] // nil if not present — emitter uses defaults
				switch lang {
				case "go":
					cfg, err := golang.ParseConfig(rawCfg, reg.Package.Go, registryPath)
					if err != nil {
						fmt.Fprintln(errOut, err)
						return errGenerationFailed
					}
					if !filepath.IsAbs(cfg.Out) {
						cfg.Out = filepath.Join(registryPath, cfg.Out)
					}
					if err := golang.Emit(reg, lock, cfg); err != nil {
						fmt.Fprintln(errOut, err)
						return errGenerationFailed
					}
					emitted = append(emitted, "go")
				case "python":
					cfg, err := python.ParseConfig(rawCfg, reg.Package.Python, registryPath)
					if err != nil {
						fmt.Fprintln(errOut, err)
						return errGenerationFailed
					}
					if !filepath.IsAbs(cfg.Out) {
						cfg.Out = filepath.Join(registryPath, cfg.Out)
					}
					if err := python.Emit(reg, lock, cfg); err != nil {
						fmt.Fprintln(errOut, err)
						return errGenerationFailed
					}
					emitted = append(emitted, "python")
				default:
					fmt.Fprintf(errOut, "unsupported language %q in codegen.languages\n", lang)
					return errGenerationFailed
				}
			}

			if len(emitted) > 0 {
				fmt.Fprintf(out, "ok: generated bindings for %s\n", strings.Join(emitted, ", "))
			} else {
				fmt.Fprintf(out, "ok: generated proto schema in %s\n", protoOut)
			}
			return nil
		},
	}
	// Hidden debug subcommands — not shown in user-facing help.
	protoCmd := newGenerateProtoCommand(out, errOut)
	protoCmd.Hidden = true
	cmd.AddCommand(protoCmd)

	constantsCmd := newGenerateConstantsCommand(out, errOut)
	constantsCmd.Hidden = true
	cmd.AddCommand(constantsCmd)

	return cmd
}

func newGenerateProtoCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "proto <registry-path> <output-dir>",
		Short: "Generate protobuf+Buf output from an OpenEvents registry",
		Long:  "Generate protobuf+Buf output from an OpenEvents registry. Downstream language code is produced by protoc plugins (e.g. protoc-gen-go, protoc-gen-python) via Buf.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, _, err := loadValidatedRegistry(args[0])
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}

			lock, err := readLockFile(lockFilePath(args[0]))
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
			if err := protogen.Render(ir, args[1]); err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}
			fmt.Fprintf(out, "ok: generated proto schema in %s\n", args[1])
			return nil
		},
	}
}

func newGenerateConstantsCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	var (
		goOut     string
		goPackage string
		pyOut     string
	)
	cmd := &cobra.Command{
		Use:   "constants <registry-path>",
		Short: "Generate cross-language event-name constants from an OpenEvents registry",
		Long: `Generate cross-language event-name constants from an OpenEvents registry.

The canonical "<name>@<version>" wire strings live in the registry. This command
emits them as Go and/or Python constants so producers and consumers in either
language can reference them without re-encoding the strings by hand.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if goOut == "" && pyOut == "" {
				fmt.Fprintln(errOut, "at least one of --go-out or --python-out is required")
				return errGenerationFailed
			}
			if goOut != "" && goPackage == "" {
				fmt.Fprintln(errOut, "--go-package is required when --go-out is set")
				return errGenerationFailed
			}

			reg, _, err := loadValidatedRegistry(args[0])
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errGenerationFailed
			}

			entries := constgen.Entries(reg)

			if goOut != "" {
				body, err := constgen.RenderGo(goPackage, entries)
				if err != nil {
					fmt.Fprintln(errOut, err)
					return errGenerationFailed
				}
				if err := writeFileAtomic(goOut, body); err != nil {
					fmt.Fprintln(errOut, err)
					return errGenerationFailed
				}
				fmt.Fprintf(out, "ok: wrote Go constants to %s\n", goOut)
			}

			if pyOut != "" {
				body, err := constgen.RenderPython(entries)
				if err != nil {
					fmt.Fprintln(errOut, err)
					return errGenerationFailed
				}
				if err := writeFileAtomic(pyOut, body); err != nil {
					fmt.Fprintln(errOut, err)
					return errGenerationFailed
				}
				fmt.Fprintf(out, "ok: wrote Python constants to %s\n", pyOut)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&goOut, "go-out", "", "path to write Go constants file")
	cmd.Flags().StringVar(&goPackage, "go-package", "", "Go package name for generated constants file")
	cmd.Flags().StringVar(&pyOut, "python-out", "", "path to write Python constants file")
	return cmd
}

func writeFileAtomic(path string, body []byte) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
