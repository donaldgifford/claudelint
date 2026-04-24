package artifact

import (
	"os"
	"sort"
	"testing"
)

func TestParseMCPFileStandalone(t *testing.T) {
	src, err := os.ReadFile("testdata/ok/mcp/standalone.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	servers, perr := ParseMCPFile(".mcp.json", src)
	if perr != nil {
		t.Fatalf("ParseMCPFile: %v", perr)
	}
	if got := len(servers); got != 3 {
		t.Fatalf("len(servers) = %d, want 3", got)
	}

	// Index by name for stable assertions.
	byName := make(map[string]*MCPServer, len(servers))
	names := make([]string, 0, len(servers))
	for _, s := range servers {
		byName[s.Name] = s
		names = append(names, s.Name)
	}
	sort.Strings(names)
	wantNames := []string{"disabled-one", "filesystem", "puppeteer"}
	for i, want := range wantNames {
		if names[i] != want {
			t.Errorf("server name[%d] = %q, want %q", i, names[i], want)
		}
	}

	pup := byName["puppeteer"]
	if pup.Command != "npx" {
		t.Errorf("puppeteer command = %q", pup.Command)
	}
	if len(pup.Args) != 2 || pup.Args[0] != "-y" {
		t.Errorf("puppeteer args = %v", pup.Args)
	}
	if pup.Env["DEBUG"] != "1" {
		t.Errorf("puppeteer env = %v", pup.Env)
	}
	if pup.Embedded {
		t.Errorf("puppeteer Embedded = true, want false (standalone file)")
	}

	if dis := byName["disabled-one"]; !dis.Disabled {
		t.Errorf("disabled-one Disabled = false, want true")
	}
}

func TestParseMCPEmbedded(t *testing.T) {
	src, err := os.ReadFile("testdata/ok/mcp/embedded_in_plugin.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	servers, err := ParseMCPEmbedded("plugin.json", src)
	if err != nil {
		t.Fatalf("ParseMCPEmbedded: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("len(servers) = %d, want 1", len(servers))
	}
	s := servers[0]
	if s.Name != "weather" {
		t.Errorf("Name = %q", s.Name)
	}
	if s.Command != "uvx" {
		t.Errorf("Command = %q", s.Command)
	}
	if !s.Embedded {
		t.Errorf("Embedded = false, want true")
	}
}

func TestParseMCPMissingServersField(t *testing.T) {
	src := []byte(`{}`)
	servers, perr := ParseMCPFile(".mcp.json", src)
	if perr != nil {
		t.Fatalf("missing servers field should not error: %v", perr)
	}
	if servers != nil {
		t.Errorf("want nil, got %v", servers)
	}
}

func TestParseMCPNonObjectServers(t *testing.T) {
	src, err := os.ReadFile("testdata/bad/mcp_nonobject_servers.json")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if _, perr := ParseMCPFile(".mcp.json", src); perr == nil {
		t.Fatal("expected ParseError for non-object servers")
	}
}

func TestParseMCPTolerantMissingCommand(t *testing.T) {
	src, err := os.ReadFile("testdata/bad/mcp_missing_command.json")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	servers, perr := ParseMCPFile(".mcp.json", src)
	if perr != nil {
		t.Fatalf("missing command field should not be a parse error: %v", perr)
	}
	if len(servers) != 1 {
		t.Fatalf("len(servers) = %d, want 1", len(servers))
	}
	if servers[0].Command != "" {
		t.Errorf("Command = %q, want empty", servers[0].Command)
	}
}

func TestParseMCPEmbeddedAbsent(t *testing.T) {
	src := []byte(`{"name":"x","version":"1.0.0"}`)
	servers, err := ParseMCPEmbedded("plugin.json", src)
	if err != nil {
		t.Fatalf("absent mcp.servers should not error: %v", err)
	}
	if servers != nil {
		t.Errorf("want nil, got %v", servers)
	}
}
