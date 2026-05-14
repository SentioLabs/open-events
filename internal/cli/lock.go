package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/sentiolabs/open-events/internal/registry"
	"github.com/sentiolabs/open-events/internal/schemair"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var errLockFailed = errors.New("lock failed")

func newLockCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "lock", Short: "Manage schema lock files"}
	cmd.AddCommand(newLockUpdateCommand(out, errOut))
	cmd.AddCommand(newLockCheckCommand(out, errOut))
	return cmd
}

func newLockUpdateCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "update <registry-path>",
		Short: "Write or update openevents.lock.yaml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, lockPath, err := loadValidatedRegistry(args[0])
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}

			existing, err := readLockFile(lockPath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}

			updated, err := schemair.UpdateLock(existing, reg)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}
			if err := writeLockFile(lockPath, updated); err != nil {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}
			fmt.Fprintf(out, "ok: updated schema lock at %s\n", lockPath)
			return nil
		},
	}
}

func newLockCheckCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "check <registry-path>",
		Short: "Check openevents.lock.yaml is current",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, lockPath, err := loadValidatedRegistry(args[0])
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}
			currentBytes, err := os.ReadFile(lockPath)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}
			lock, err := decodeLockFile(currentBytes)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}
			if err := schemair.CheckLock(lock, reg); err != nil {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}
			updated, err := schemair.UpdateLock(lock, reg)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}
			expectedBytes, err := marshalLockFile(updated)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return errLockFailed
			}
			if !bytes.Equal(currentBytes, expectedBytes) {
				fmt.Fprintf(errOut, "schema lock is not canonical at %s; run `openevents lock update %s`\n", lockPath, args[0])
				return errLockFailed
			}
			fmt.Fprintf(out, "ok: schema lock is current at %s\n", lockPath)
			return nil
		},
	}
}

func loadValidatedRegistry(path string) (registry.Registry, string, error) {
	reg, loadDiags := registry.Load(path)
	if loadDiags.HasErrors() {
		return registry.Registry{}, "", errors.New(loadDiags.Error())
	}
	validationDiags := registry.Validate(reg)
	if validationDiags.HasErrors() {
		return registry.Registry{}, "", errors.New(validationDiags.Error())
	}
	return reg, lockFilePath(path), nil
}

func lockFilePath(registryPath string) string {
	return filepath.Join(registryRootPath(registryPath), "openevents.lock.yaml")
}

func registryRootPath(registryPath string) string {
	info, err := os.Stat(registryPath)
	if err == nil && info.IsDir() {
		return registryPath
	}
	return filepath.Dir(registryPath)
}

func readLockFile(path string) (schemair.Lock, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return schemair.Lock{}, err
	}
	return decodeLockFile(content)
}

func decodeLockFile(content []byte) (schemair.Lock, error) {
	var file lockFile
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil {
		return schemair.Lock{}, err
	}
	return file.toSchemaLock(), nil
}

func writeLockFile(path string, lock schemair.Lock) error {
	content, err := marshalLockFile(lock)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func marshalLockFile(lock schemair.Lock) ([]byte, error) {
	return yaml.Marshal(newLockFile(lock))
}

type lockFile struct {
	Version int                         `yaml:"version"`
	Domains map[string]lockDomainFields `yaml:"domains"`
	Events  map[string]lockEventFields  `yaml:"events"`
}

type lockDomainFields struct {
	Context  map[string]lockField `yaml:"context"`
	Reserved []lockReservedField  `yaml:"reserved,omitempty"`
}

type lockEventFields struct {
	Envelope   map[string]lockField `yaml:"envelope"`
	Properties map[string]lockField `yaml:"properties"`
	Reserved   []lockReservedField  `yaml:"reserved"`
}

type lockField struct {
	StableID    string                 `yaml:"stable_id"`
	ProtoNumber int                    `yaml:"proto_number"`
	Properties  map[string]lockField   `yaml:"properties,omitempty"`
	Reserved    []lockReservedField    `yaml:"reserved,omitempty"`
}

type lockReservedField struct {
	Name        string `yaml:"name"`
	StableID    string `yaml:"stable_id"`
	ProtoNumber int    `yaml:"proto_number"`
	Reason      string `yaml:"reason"`
}

// schemairFieldToLockField converts a schemair.LockedField to its YAML lockField
// representation, recursing into nested object subfields when present.
func schemairFieldToLockField(lf schemair.LockedField) lockField {
	f := lockField{StableID: lf.StableID, ProtoNumber: lf.ProtoNumber}
	if len(lf.Properties) > 0 {
		f.Properties = make(map[string]lockField, len(lf.Properties))
		for name, sub := range lf.Properties {
			f.Properties[name] = schemairFieldToLockField(sub)
		}
	}
	if len(lf.Reserved) > 0 {
		f.Reserved = make([]lockReservedField, 0, len(lf.Reserved))
		for _, rv := range lf.Reserved {
			f.Reserved = append(f.Reserved, lockReservedField{Name: rv.Name, StableID: rv.StableID, ProtoNumber: rv.ProtoNumber, Reason: rv.Reason})
		}
		sort.Slice(f.Reserved, func(i, j int) bool {
			if f.Reserved[i].ProtoNumber != f.Reserved[j].ProtoNumber {
				return f.Reserved[i].ProtoNumber < f.Reserved[j].ProtoNumber
			}
			return f.Reserved[i].Name < f.Reserved[j].Name
		})
	}
	return f
}

// lockFieldToSchemairField converts a YAML lockField back to schemair.LockedField,
// recursing into nested object subfields when present.
func lockFieldToSchemairField(f lockField) schemair.LockedField {
	lf := schemair.LockedField{StableID: f.StableID, ProtoNumber: f.ProtoNumber}
	if len(f.Properties) > 0 {
		lf.Properties = make(map[string]schemair.LockedField, len(f.Properties))
		for name, sub := range f.Properties {
			lf.Properties[name] = lockFieldToSchemairField(sub)
		}
	}
	if len(f.Reserved) > 0 {
		lf.Reserved = make([]schemair.ReservedField, 0, len(f.Reserved))
		for _, rv := range f.Reserved {
			lf.Reserved = append(lf.Reserved, schemair.ReservedField{Name: rv.Name, StableID: rv.StableID, ProtoNumber: rv.ProtoNumber, Reason: rv.Reason})
		}
	}
	return lf
}

func newLockFile(lock schemair.Lock) lockFile {
	file := lockFile{
		Version: lock.Version,
		Domains: make(map[string]lockDomainFields, len(lock.Domains)),
		Events:  make(map[string]lockEventFields, len(lock.Events)),
	}
	for domainName, domain := range lock.Domains {
		ctx := make(map[string]lockField, len(domain.Context))
		for fieldName, locked := range domain.Context {
			ctx[fieldName] = schemairFieldToLockField(locked)
		}
		domainEntry := lockDomainFields{Context: ctx}
		if len(domain.Reserved) > 0 {
			reserved := make([]lockReservedField, 0, len(domain.Reserved))
			for _, rv := range domain.Reserved {
				reserved = append(reserved, lockReservedField{Name: rv.Name, StableID: rv.StableID, ProtoNumber: rv.ProtoNumber, Reason: rv.Reason})
			}
			sort.Slice(reserved, func(i, j int) bool {
				if reserved[i].ProtoNumber != reserved[j].ProtoNumber {
					return reserved[i].ProtoNumber < reserved[j].ProtoNumber
				}
				return reserved[i].Name < reserved[j].Name
			})
			domainEntry.Reserved = reserved
		}
		file.Domains[domainName] = domainEntry
	}
	for k, v := range lock.Events {
		item := lockEventFields{
			Envelope:   make(map[string]lockField, len(v.Envelope)),
			Properties: make(map[string]lockField, len(v.Properties)),
			Reserved:   make([]lockReservedField, 0, len(v.Reserved)),
		}
		for fk, fv := range v.Envelope {
			item.Envelope[fk] = schemairFieldToLockField(fv)
		}
		for fk, fv := range v.Properties {
			item.Properties[fk] = schemairFieldToLockField(fv)
		}
		for _, rv := range v.Reserved {
			item.Reserved = append(item.Reserved, lockReservedField{Name: rv.Name, StableID: rv.StableID, ProtoNumber: rv.ProtoNumber, Reason: rv.Reason})
		}
		sort.Slice(item.Reserved, func(i, j int) bool {
			if item.Reserved[i].ProtoNumber != item.Reserved[j].ProtoNumber {
				return item.Reserved[i].ProtoNumber < item.Reserved[j].ProtoNumber
			}
			return item.Reserved[i].Name < item.Reserved[j].Name
		})
		file.Events[k] = item
	}
	return file
}

func (f lockFile) toSchemaLock() schemair.Lock {
	lock := schemair.Lock{
		Version: f.Version,
		Domains: make(map[string]schemair.LockedDomain, len(f.Domains)),
		Events:  make(map[string]schemair.LockedEvent, len(f.Events)),
	}
	for domainName, domain := range f.Domains {
		ctx := make(map[string]schemair.LockedField, len(domain.Context))
		for fieldName, locked := range domain.Context {
			ctx[fieldName] = lockFieldToSchemairField(locked)
		}
		reserved := make([]schemair.ReservedField, 0, len(domain.Reserved))
		for _, rv := range domain.Reserved {
			reserved = append(reserved, schemair.ReservedField{Name: rv.Name, StableID: rv.StableID, ProtoNumber: rv.ProtoNumber, Reason: rv.Reason})
		}
		lock.Domains[domainName] = schemair.LockedDomain{Context: ctx, Reserved: reserved}
	}
	for k, v := range f.Events {
		event := schemair.LockedEvent{
			Envelope:   make(map[string]schemair.LockedField, len(v.Envelope)),
			Properties: make(map[string]schemair.LockedField, len(v.Properties)),
			Reserved:   make([]schemair.ReservedField, 0, len(v.Reserved)),
		}
		for fk, fv := range v.Envelope {
			event.Envelope[fk] = lockFieldToSchemairField(fv)
		}
		for fk, fv := range v.Properties {
			event.Properties[fk] = lockFieldToSchemairField(fv)
		}
		for _, rv := range v.Reserved {
			event.Reserved = append(event.Reserved, schemair.ReservedField{Name: rv.Name, StableID: rv.StableID, ProtoNumber: rv.ProtoNumber, Reason: rv.Reason})
		}
		lock.Events[k] = event
	}
	return lock
}
