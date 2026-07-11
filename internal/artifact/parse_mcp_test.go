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

	// standalone.json uses the deprecated top-level servers{} key.
	for _, s := range servers {
		if !s.LegacyServersKey {
			t.Errorf("server %q LegacyServersKey = false, want true (legacy servers{} file)", s.Name)
		}
	}
}

func TestParseMCPFileMCPServersKey(t *testing.T) {
	src, err := os.ReadFile("testdata/ok/mcp/standalone_mcpservers.json")
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
	byName := make(map[string]*MCPServer, len(servers))
	for _, s := range servers {
		byName[s.Name] = s
		if s.LegacyServersKey {
			t.Errorf("server %q LegacyServersKey = true, want false (mcpServers{} file)", s.Name)
		}
	}
	gh := byName["github"]
	if gh == nil {
		t.Fatal("server github not parsed")
	}
	if gh.Command != "npx" {
		t.Errorf("github command = %q, want npx", gh.Command)
	}
	if gh.Env["DEBUG"] != "1" {
		t.Errorf("github env = %v", gh.Env)
	}
	if gh.Transport != "" {
		t.Errorf("github Transport = %q, want empty (type omitted)", gh.Transport)
	}
	if got := gh.EffectiveTransport(); got != "stdio" {
		t.Errorf("github EffectiveTransport() = %q, want stdio", got)
	}

	lin := byName["linear"]
	if lin == nil {
		t.Fatal("server linear not parsed")
	}
	if lin.Transport != "http" {
		t.Errorf("linear Transport = %q, want http", lin.Transport)
	}
	if lin.TransportRange.IsZero() {
		t.Error("linear TransportRange is zero, want a real range")
	}
	if lin.URL != "https://mcp.linear.app/mcp" {
		t.Errorf("linear URL = %q", lin.URL)
	}
	if lin.URLRange.IsZero() {
		t.Error("linear URLRange is zero, want a real range")
	}
	if lin.Headers["X-Team"] != "platform" {
		t.Errorf("linear Headers = %v", lin.Headers)
	}
	if lin.HeadersHelper != "./scripts/mint-token.sh" {
		t.Errorf("linear HeadersHelper = %q", lin.HeadersHelper)
	}
	if lin.TimeoutMS != 30000 {
		t.Errorf("linear TimeoutMS = %d, want 30000", lin.TimeoutMS)
	}
	if !lin.AlwaysLoad {
		t.Error("linear AlwaysLoad = false, want true")
	}
	if !lin.HasOAuth {
		t.Error("linear HasOAuth = false, want true")
	}
	if lin.Command != "" {
		t.Errorf("linear Command = %q, want empty (remote transport)", lin.Command)
	}
}

func TestParseMCPFileBothKeysMCPServersWins(t *testing.T) {
	src := []byte(`{
		"mcpServers": {"primary": {"command": "npx"}},
		"servers": {"legacy": {"command": "uvx"}}
	}`)
	servers, perr := ParseMCPFile(".mcp.json", src)
	if perr != nil {
		t.Fatalf("ParseMCPFile: %v", perr)
	}
	if len(servers) != 1 {
		t.Fatalf("len(servers) = %d, want 1 (mcpServers wins)", len(servers))
	}
	if servers[0].Name != "primary" {
		t.Errorf("Name = %q, want primary", servers[0].Name)
	}
	if servers[0].LegacyServersKey {
		t.Error("LegacyServersKey = true, want false when mcpServers wins")
	}
}

func TestParseMCPNonObjectMCPServers(t *testing.T) {
	src := []byte(`{"mcpServers": ["not", "an", "object"]}`)
	_, perr := ParseMCPFile(".mcp.json", src)
	if perr == nil {
		t.Fatal("expected ParseError for non-object mcpServers")
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
