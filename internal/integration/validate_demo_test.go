package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAndGenerateDemoRegistry(t *testing.T) {
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
}
