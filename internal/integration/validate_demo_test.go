package integration_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runCommand(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}

func ensureBuf(t *testing.T) string {
	t.Helper()

	bufPath := filepath.Join("..", "..", ".tools", "bin", "buf")
	protocGenGoPath := filepath.Join("..", "..", ".tools", "bin", "protoc-gen-go")
	if _, err := os.Stat(bufPath); err != nil {
		if os.IsNotExist(err) {
			runCommand(t, "", nil, "bash", "../../scripts/install-buf.sh")
		} else {
			t.Fatalf("stat %s: %v", bufPath, err)
		}
	}
	if _, err := os.Stat(protocGenGoPath); err != nil {
		if os.IsNotExist(err) {
			runCommand(t, "", nil, "bash", "../../scripts/install-buf.sh")
		} else {
			t.Fatalf("stat %s: %v", protocGenGoPath, err)
		}
	}

	absPath, err := filepath.Abs(bufPath)
	if err != nil {
		t.Fatalf("abs %s: %v", bufPath, err)
	}
	return absPath
}

var (
	errPythonProtobufVersionMismatch = errors.New("python protobuf version mismatch")
	errPythonProtobufPathMismatch    = errors.New("python protobuf path mismatch")
	errPythonProtobufOutputFormat    = errors.New("python protobuf output format mismatch")
)

func validatePinnedPythonRuntime(expectedVersion string, pythonTools string, output string) error {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		return errPythonProtobufOutputFormat
	}
	if lines[0] != expectedVersion {
		return fmt.Errorf("%w: got %q want %q", errPythonProtobufVersionMismatch, lines[0], expectedVersion)
	}
	protobufPath := filepath.Clean(lines[1])
	if !strings.HasPrefix(protobufPath, filepath.Clean(pythonTools)+string(filepath.Separator)) {
		return fmt.Errorf("%w: got %q, want under %q", errPythonProtobufPathMismatch, protobufPath, pythonTools)
	}
	return nil
}

func ensurePinnedPythonPath(t *testing.T, out string) string {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("abs repo root: %v", err)
	}
	pythonTools := filepath.Join(repoRoot, ".tools", "python-protobuf")
	protobufInit := filepath.Join(pythonTools, "google", "protobuf", "__init__.py")
	versionFile := filepath.Join(repoRoot, ".tools", "python-protobuf.version")
	versionBytes, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("read %s: %v", versionFile, err)
	}
	expectedVersion := strings.TrimSpace(string(versionBytes))
	if expectedVersion == "" {
		t.Fatalf("missing python protobuf version in %s", versionFile)
	}

	checkRuntime := func(pythonPath string, script string) ([]byte, error) {
		cmd := exec.Command("python3", "-c", script)
		cmd.Env = append(os.Environ(), "PYTHONPATH="+pythonPath)
		return cmd.CombinedOutput()
	}

	if _, err := os.Stat(protobufInit); err != nil {
		if os.IsNotExist(err) {
			runCommand(t, "", nil, "bash", "../../scripts/install-buf.sh")
		} else {
			t.Fatalf("stat %s: %v", protobufInit, err)
		}
	}

	runtimeScript := "import google.protobuf; print(google.protobuf.__version__); print(google.protobuf.__file__)"
	checkOutput, err := checkRuntime(pythonTools, runtimeScript)
	if err != nil || validatePinnedPythonRuntime(expectedVersion, pythonTools, string(checkOutput)) != nil {
		runCommand(t, "", nil, "bash", "../../scripts/install-buf.sh")
		checkOutput, err = checkRuntime(pythonTools, runtimeScript)
		if err != nil {
			t.Fatalf("verify local python protobuf runtime: %v\n%s", err, checkOutput)
		}
		if err := validatePinnedPythonRuntime(expectedVersion, pythonTools, string(checkOutput)); err != nil {
			t.Fatalf("verify local python protobuf runtime provenance: %v\n%s", err, checkOutput)
		}
	}

	pythonGenerated := filepath.Join(out, "gen", "python")
	compatScript := "from com.acme.storefront.v1 import events_pb2; import google.protobuf; print(google.protobuf.__version__); print(google.protobuf.__file__)"
	compatOutput, err := checkRuntime(fmt.Sprintf("%s%c%s", pythonTools, os.PathListSeparator, pythonGenerated), compatScript)
	if err != nil {
		t.Fatalf("verify pinned python protobuf compatibility: %v\n%s", err, compatOutput)
	}
	if err := validatePinnedPythonRuntime(expectedVersion, pythonTools, string(compatOutput)); err != nil {
		t.Fatalf("compat runtime verification failed: %v\n%s", err, compatOutput)
	}

	return fmt.Sprintf("%s%c%s", pythonTools, os.PathListSeparator, pythonGenerated)
}

func copyDir(t *testing.T, src string, dst string) {
	t.Helper()

	if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode().Perm())
	}); err != nil {
		t.Fatalf("copy %s to %s: %v", src, dst, err)
	}
}

func TestBufGeneratedGoPythonInterop(t *testing.T) {
	t.Helper()

	temp := t.TempDir()
	demoCopy := filepath.Join(temp, "demo")
	copyDir(t, "../../examples/demo/registry", demoCopy)

	runCommand(t, "", nil, "go", "run", "../../cmd/openevents", "lock", "update", demoCopy)
	runCommand(t, "", nil, "go", "run", "../../cmd/openevents", "lock", "check", demoCopy)

	out := filepath.Join(temp, "proto-out")
	runCommand(t, "", nil, "go", "run", "../../cmd/openevents", "generate", "proto", demoCopy, out)

	bufPath := ensureBuf(t)
	toolsBin := filepath.Dir(bufPath)
	bufEnv := []string{"PATH=" + toolsBin + string(os.PathListSeparator) + os.Getenv("PATH")}
	runCommand(t, out, bufEnv, bufPath, "generate", ".")

	goGenerated := filepath.Join(out, "gen", "go", "com", "acme", "storefront", "v1", "events.pb.go")
	pythonGenerated := filepath.Join(out, "gen", "python", "com", "acme", "storefront", "v1", "events_pb2.py")
	if _, err := os.Stat(goGenerated); err != nil {
		t.Fatalf("expected generated file %s to exist: %v", goGenerated, err)
	}
	if _, err := os.Stat(pythonGenerated); err != nil {
		t.Fatalf("expected generated file %s to exist: %v", pythonGenerated, err)
	}

	goModuleDir := filepath.Join(temp, "bufinterop")
	if err := os.MkdirAll(filepath.Join(goModuleDir, "generated"), 0o755); err != nil {
		t.Fatalf("mkdir go module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goModuleDir, "go.mod"), []byte("module bufinterop\n\ngo 1.24\n\nrequire google.golang.org/protobuf v1.36.6\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	generatedPBContent, err := os.ReadFile(goGenerated)
	if err != nil {
		t.Fatalf("read generated go protobuf: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goModuleDir, "generated", "events.pb.go"), generatedPBContent, 0o644); err != nil {
		t.Fatalf("write copied go protobuf: %v", err)
	}

	payloadFile := filepath.Join(temp, "event.pb")
	goProgram := `package main

import (
	"os"

	events "bufinterop/generated"
	"google.golang.org/protobuf/proto"
)

func main() {
	e := &events.CheckoutCompletedV1{
		EventName:    "checkout.completed",
		EventVersion: 1,
		Context: &events.Context{
			TenantId: proto.String("tenant-42"),
		},
		Properties: &events.CheckoutCompletedV1Properties{
			OrderId:    proto.String("ord-001"),
			TotalCents: proto.Int64(1099),
		},
	}

	b, err := proto.Marshal(e)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[1], b, 0o644); err != nil {
		panic(err)
	}
}
`
	if err := os.WriteFile(filepath.Join(goModuleDir, "main.go"), []byte(goProgram), 0o644); err != nil {
		t.Fatalf("write go interop program: %v", err)
	}

	runCommand(t, goModuleDir, nil, "go", "mod", "tidy")
	runCommand(t, goModuleDir, nil, "go", "run", ".", payloadFile)

	pyScript := `import sys
from com.acme.storefront.v1 import events_pb2

with open(sys.argv[1], "rb") as handle:
    payload = handle.read()

event = events_pb2.CheckoutCompletedV1()
event.ParseFromString(payload)

assert event.event_name == "checkout.completed"
assert event.event_version == 1
assert event.context.tenant_id == "tenant-42"
assert event.properties.order_id == "ord-001"
assert event.properties.total_cents == 1099
`
	pyScriptPath := filepath.Join(temp, "check_buf_event.py")
	if err := os.WriteFile(pyScriptPath, []byte(pyScript), 0o644); err != nil {
		t.Fatalf("write python interop script: %v", err)
	}

	pythonPath := ensurePinnedPythonPath(t, out)
	runCommand(t, "", []string{"PYTHONPATH=" + pythonPath}, "python3", pyScriptPath, payloadFile)
}

func TestValidatePinnedPythonRuntime(t *testing.T) {
	repoRoot := filepath.Clean(string(filepath.Separator) + filepath.Join("repo"))
	pythonTools := filepath.Join(repoRoot, ".tools", "python-protobuf")

	tests := []struct {
		name    string
		output  string
		wantErr error
	}{
		{
			name:    "accepts expected version and local file",
			output:  "1.2.3\n" + filepath.Join(pythonTools, "google", "protobuf", "__init__.py"),
			wantErr: nil,
		},
		{
			name:    "rejects wrong version",
			output:  "9.9.9\n" + filepath.Join(pythonTools, "google", "protobuf", "__init__.py"),
			wantErr: errPythonProtobufVersionMismatch,
		},
		{
			name:    "rejects global fallback",
			output:  "1.2.3\n/usr/lib/python3/dist-packages/google/protobuf/__init__.py",
			wantErr: errPythonProtobufPathMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePinnedPythonRuntime("1.2.3", pythonTools, tt.output)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validatePinnedPythonRuntime() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDemoExampleHasCommittedLock(t *testing.T) {
	lockCheck := exec.Command("go", "run", "../../cmd/openevents", "lock", "check", "../../examples/demo/registry")
	lockCheckOut, err := lockCheck.CombinedOutput()
	if err != nil {
		t.Fatalf("lock check demo failed: %v\n%s", err, lockCheckOut)
	}
}

func TestValidateAndGenerateDemoRegistry(t *testing.T) {
	t.Run("validate_demo", func(t *testing.T) {
		validate := exec.Command("go", "run", "../../cmd/openevents", "validate", "../../examples/demo/registry")
		validateOut, err := validate.CombinedOutput()
		if err != nil {
			t.Fatalf("validate demo failed: %v\n%s", err, validateOut)
		}

		if got, want := strings.TrimSpace(string(validateOut)), "ok: registry valid (3 events, 4 context fields)"; got != want {
			t.Fatalf("validate output = %q, want %q", got, want)
		}
	})

	t.Run("generate_demo_proto", func(t *testing.T) {
		temp := t.TempDir()
		demoCopy := filepath.Join(temp, "demo")
		copyDir(t, "../../examples/demo/registry", demoCopy)

		runCommand(t, "", nil, "go", "run", "../../cmd/openevents", "lock", "update", demoCopy)
		runCommand(t, "", nil, "go", "run", "../../cmd/openevents", "lock", "check", demoCopy)

		out := filepath.Join(temp, "proto-out")
		runCommand(t, "", nil, "go", "run", "../../cmd/openevents", "generate", "proto", demoCopy, out)

		for _, rel := range []string{
			"buf.yaml",
			"buf.gen.yaml",
			"openevents.metadata.yaml",
			"proto/com/acme/storefront/v1/events.proto",
		} {
			if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
				t.Fatalf("expected %s to exist: %v", rel, err)
			}
		}

		bufPath := ensureBuf(t)
		toolsBin := filepath.Dir(bufPath)
		bufEnv := []string{"PATH=" + toolsBin + string(os.PathListSeparator) + os.Getenv("PATH")}
		runCommand(t, "", nil, bufPath, "lint", out)
		runCommand(t, "", nil, bufPath, "build", out)
		runCommand(t, out, bufEnv, bufPath, "generate", ".")

		for _, rel := range []string{
			"gen/go/com/acme/storefront/v1/events.pb.go",
			"gen/python/com/acme/storefront/v1/events_pb2.py",
		} {
			if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
				t.Fatalf("expected generated file %s to exist: %v", rel, err)
			}
		}
	})
}
