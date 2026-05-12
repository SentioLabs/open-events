package integration_test

import (
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

func runBufGeneratedGoPythonInterop(t *testing.T) {
	t.Helper()

	temp := t.TempDir()
	demoCopy := filepath.Join(temp, "demo")
	copyDir(t, "../../examples/demo", demoCopy)

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

	pythonPath := filepath.Join(out, "gen", "python")
	if _, err := exec.Command("python3", "-c", "import google.protobuf").CombinedOutput(); err != nil {
		pythonDeps := filepath.Join(temp, "pydeps")
		runCommand(t, "", nil, "python3", "-m", "pip", "install", "--quiet", "--target", pythonDeps, "protobuf")
		pythonPath = pythonDeps + string(os.PathListSeparator) + pythonPath
	}

	runCommand(t, "", []string{"PYTHONPATH=" + pythonPath}, "python3", pyScriptPath, payloadFile)
}

func TestValidateAndGenerateDemoRegistry(t *testing.T) {
	t.Run("go_python", func(t *testing.T) {
		validate := exec.Command("go", "run", "../../cmd/openevents", "validate", "../../examples/demo")
		validateOut, err := validate.CombinedOutput()
		if err != nil {
			t.Fatalf("validate demo failed: %v\n%s", err, validateOut)
		}

		if got, want := strings.TrimSpace(string(validateOut)), "ok: registry valid (3 events, 4 context fields)"; got != want {
			t.Fatalf("validate output = %q, want %q", got, want)
		}

		temp := t.TempDir()
		goModuleDir := filepath.Join(temp, "storefront")
		if err := os.MkdirAll(goModuleDir, 0o755); err != nil {
			t.Fatalf("mkdir module dir: %v", err)
		}

		if err := os.WriteFile(filepath.Join(goModuleDir, "go.mod"), []byte("module github.com/acme/storefront\n\ngo 1.24\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}

		goGenerate := exec.Command("go", "run", "../../cmd/openevents", "generate", "go", "../../examples/demo", filepath.Join(goModuleDir, "events"))
		goGenerateOut, err := goGenerate.CombinedOutput()
		if err != nil {
			t.Fatalf("generate go failed: %v\n%s", err, goGenerateOut)
		}

		payloadFile := filepath.Join(temp, "event.json")
		goProgram := `package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/acme/storefront/events"
)

func main() {
	coupon := "SAVE10"
	e := events.NewCheckoutCompletedV1(
		"evt-123",
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		events.Client{Name: "storefront-api", Version: "1.0.0"},
		events.Context{TenantID: "tenant-42", Platform: events.PlatformWeb},
		events.CheckoutCompletedV1Properties{
			OrderID: "ord-001",
			CartID: "cart-001",
			PaymentMethod: events.PaymentMethodCard,
			TotalCents: 1099,
			CouponCode: &coupon,
		},
	)
	b, err := json.Marshal(e)
	if err != nil { panic(err) }
	if err := os.WriteFile(os.Args[1], b, 0o644); err != nil { panic(err) }
}
`
		if err := os.WriteFile(filepath.Join(goModuleDir, "main.go"), []byte(goProgram), 0o644); err != nil {
			t.Fatalf("write go program: %v", err)
		}

		runGo := exec.Command("go", "run", ".", payloadFile)
		runGo.Dir = goModuleDir
		runGoOut, err := runGo.CombinedOutput()
		if err != nil {
			t.Fatalf("run generated go program failed: %v\n%s", err, runGoOut)
		}

		pythonOut := filepath.Join(temp, "python")
		pyGenerate := exec.Command("go", "run", "../../cmd/openevents", "generate", "python", "../../examples/demo", pythonOut)
		pyGenerateOut, err := pyGenerate.CombinedOutput()
		if err != nil {
			t.Fatalf("generate python failed: %v\n%s", err, pyGenerateOut)
		}

		pyScript := `import json
from acme_storefront.events import decode_event

with open(__import__('sys').argv[1], 'r', encoding='utf-8') as f:
    event = decode_event(json.load(f))

assert event.event_name == 'checkout.completed'
assert event.event_version == 1
assert event.context.tenant_id == 'tenant-42'
assert event.context.platform == 'web'
assert event.properties.order_id == 'ord-001'
assert event.properties.total_cents == 1099
`
		pyScriptPath := filepath.Join(temp, "check_event.py")
		if err := os.WriteFile(pyScriptPath, []byte(pyScript), 0o644); err != nil {
			t.Fatalf("write python script: %v", err)
		}

		runPy := exec.Command("python3", pyScriptPath, payloadFile)
		runPy.Env = append(os.Environ(), "PYTHONPATH="+pythonOut)
		runPyOut, err := runPy.CombinedOutput()
		if err != nil {
			t.Fatalf("run generated python script failed: %v\n%s", err, runPyOut)
		}
	})

	t.Run("buf_go_python_interop", func(t *testing.T) {
		runBufGeneratedGoPythonInterop(t)
	})

	t.Run("TestValidateAndGenerateDemoProto", func(t *testing.T) {
		temp := t.TempDir()
		demoCopy := filepath.Join(temp, "demo")
		copyDir(t, "../../examples/demo", demoCopy)

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
