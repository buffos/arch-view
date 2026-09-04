package layout

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"
)

// SC-AER-008 verifies the published catalog against the shipped ELK binary,
// not an approximation of its options or a mock layout engine.
func TestPinnedUsefulSettingsRuntime(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable under when-supported verification policy")
	}
	catalog, err := json.Marshal(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(node, "../web/layout_settings_fixture.cjs")
	command.Stdin = bytes.NewReader(catalog)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
}
