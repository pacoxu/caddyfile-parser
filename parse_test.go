package caddyfileparser

import (
	"strings"
	"testing"
)

// ---- Caddyfile / CoreDNS parser tests ----------------------------------------

func TestParse_CaddyfileExample(t *testing.T) {
	input := `label1 {
	directive1 arg1
	directive2 arg2 {
	    subdir1 arg3 arg4
	    subdir2
	    # nested blocks not supported
	}
	directive3
}`

	blocks, err := Parse("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	assertKeys(t, b.Keys, "label1")

	if len(b.Segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(b.Segments))
	}

	// directive1 arg1
	assertSegment(t, b.Segments[0], "directive1", []string{"arg1"}, 0)

	// directive2 arg2 { subdir1 arg3 arg4 / subdir2 }
	seg2 := b.Segments[1]
	assertSegment(t, seg2, "directive2", []string{"arg2"}, 2)
	assertSegment(t, seg2.Block[0], "subdir1", []string{"arg3", "arg4"}, 0)
	assertSegment(t, seg2.Block[1], "subdir2", nil, 0)

	// directive3
	assertSegment(t, b.Segments[2], "directive3", nil, 0)
}

func TestParse_CorefileExample(t *testing.T) {
	input := `.:53 {
    errors
    health
    kubernetes cluster.local in-addr.arpa ip6.arpa {
       pods insecure
       upstream
       fallthrough in-addr.arpa ip6.arpa
    }
    prometheus :9153
    cache 30
}`

	blocks, err := Parse("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	assertKeys(t, b.Keys, ".:53")

	if len(b.Segments) != 5 {
		t.Fatalf("expected 5 segments, got %d: %v", len(b.Segments), segNames(b.Segments))
	}

	assertSegment(t, b.Segments[0], "errors", nil, 0)
	assertSegment(t, b.Segments[1], "health", nil, 0)

	kube := b.Segments[2]
	assertSegment(t, kube, "kubernetes", []string{"cluster.local", "in-addr.arpa", "ip6.arpa"}, 3)
	assertSegment(t, kube.Block[0], "pods", []string{"insecure"}, 0)
	assertSegment(t, kube.Block[1], "upstream", nil, 0)
	assertSegment(t, kube.Block[2], "fallthrough", []string{"in-addr.arpa", "ip6.arpa"}, 0)

	assertSegment(t, b.Segments[3], "prometheus", []string{":9153"}, 0)
	assertSegment(t, b.Segments[4], "cache", []string{"30"}, 0)
}

func TestParse_MultipleBlocks(t *testing.T) {
	input := `example.com {
    root * /var/www/html
}

api.example.com {
    reverse_proxy localhost:8080
}`

	blocks, err := Parse("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	assertKeys(t, blocks[0].Keys, "example.com")
	assertKeys(t, blocks[1].Keys, "api.example.com")
}

func TestParse_MultipleKeys(t *testing.T) {
	input := `localhost example.com :8080 {
    respond "hello"
}`

	blocks, err := Parse("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if len(b.Keys) != 3 {
		t.Fatalf("expected 3 keys, got %v", b.Keys)
	}
	if b.Keys[0] != "localhost" || b.Keys[1] != "example.com" || b.Keys[2] != ":8080" {
		t.Errorf("unexpected keys: %v", b.Keys)
	}
}

func TestParse_QuotedArgs(t *testing.T) {
	input := `site.com {
    respond "hello world"
}`

	blocks, err := Parse("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	seg := blocks[0].Segments[0]
	if len(seg.Args) != 1 || seg.Args[0] != "hello world" {
		t.Errorf("expected args [\"hello world\"], got %v", seg.Args)
	}
}

func TestParse_Comments(t *testing.T) {
	input := `# top-level comment
site.com {
    # directive comment
    gzip on # inline comment
}`

	blocks, err := Parse("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if len(b.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d: %v", len(b.Segments), segNames(b.Segments))
	}
	assertSegment(t, b.Segments[0], "gzip", []string{"on"}, 0)
}

func TestParse_Empty(t *testing.T) {
	blocks, err := Parse("test", strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestParse_ErrorMissingOpenBrace(t *testing.T) {
	input := `label1 directive`
	_, err := Parse("test", strings.NewReader(input))
	if err == nil {
		t.Error("expected error for missing opening brace, got nil")
	}
}

func TestParse_ErrorUnexpectedCloseBrace(t *testing.T) {
	input := `}`
	_, err := Parse("test", strings.NewReader(input))
	if err == nil {
		t.Error("expected error for unexpected '}', got nil")
	}
}

func TestParse_ErrorUnclosedBlock(t *testing.T) {
	input := `label1 {
    directive1`
	_, err := Parse("test", strings.NewReader(input))
	if err == nil {
		t.Error("expected error for unclosed block, got nil")
	}
}

// ---- Nginx parser tests -------------------------------------------------------

func TestParseNginx_Example(t *testing.T) {
	input := `user  root;
worker_processes  1;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;

events {
    worker_connections  65535;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    server_tokens off;

    sendfile        on;
    keepalive_timeout  18400;

    gzip  on;
    gzip_disable "msie6";

    client_max_body_size 0;

    include /etc/nginx/conf.d/*.conf;
}`

	cfg, err := ParseNginx("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseNginx error: %v", err)
	}

	if len(cfg) == 0 {
		t.Fatal("expected non-empty config")
	}

	// Check that top-level simple directives are parsed.
	assertNginxDirective(t, cfg[0], "user", []string{"root"}, false)
	assertNginxDirective(t, cfg[1], "worker_processes", []string{"1"}, false)

	// Find events block.
	eventsIdx := findNginxDirective(cfg, "events")
	if eventsIdx < 0 {
		t.Fatal("expected 'events' block")
	}
	events := cfg[eventsIdx]
	if events.Body == nil {
		t.Fatal("events block should have a body")
	}
	if len(events.Body) != 1 || events.Body[0].Name != "worker_connections" {
		t.Errorf("unexpected events body: %v", events.Body)
	}

	// Find http block.
	httpIdx := findNginxDirective(cfg, "http")
	if httpIdx < 0 {
		t.Fatal("expected 'http' block")
	}
	http := cfg[httpIdx]
	if http.Body == nil {
		t.Fatal("http block should have a body")
	}
}

func TestParseNginx_NestedBlocks(t *testing.T) {
	input := `http {
    server {
        listen 80;
        server_name example.com;
        location / {
            root /var/www/html;
        }
    }
}`

	cfg, err := ParseNginx("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseNginx error: %v", err)
	}

	if len(cfg) != 1 {
		t.Fatalf("expected 1 top-level directive, got %d", len(cfg))
	}

	http := cfg[0]
	if http.Name != "http" || http.Body == nil {
		t.Fatalf("expected http block, got %+v", http)
	}

	server := http.Body[0]
	if server.Name != "server" || server.Body == nil {
		t.Fatalf("expected server block, got %+v", server)
	}
	if len(server.Body) != 3 {
		t.Fatalf("expected 3 directives in server, got %d", len(server.Body))
	}

	loc := server.Body[2]
	if loc.Name != "location" || loc.Body == nil {
		t.Fatalf("expected location block, got %+v", loc)
	}
}

func TestParseNginx_QuotedStrings(t *testing.T) {
	input := `gzip_disable "msie6";`

	cfg, err := ParseNginx("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseNginx error: %v", err)
	}

	if len(cfg) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(cfg))
	}

	d := cfg[0]
	if d.Name != "gzip_disable" || len(d.Params) != 1 || d.Params[0] != "msie6" {
		t.Errorf("unexpected directive: %+v", d)
	}
}

func TestParseNginx_MultipleParamsOnOneLine(t *testing.T) {
	input := `log_format main '$remote_addr - $remote_user' '$status $body_bytes_sent';`

	cfg, err := ParseNginx("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseNginx error: %v", err)
	}

	if len(cfg) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(cfg))
	}

	d := cfg[0]
	if d.Name != "log_format" {
		t.Errorf("expected log_format, got %q", d.Name)
	}
	if len(d.Params) != 3 {
		t.Errorf("expected 3 params, got %v", d.Params)
	}
}

func TestParseNginx_Comments(t *testing.T) {
	input := `# main comment
sendfile on; # inline comment
#tcp_nopush on;`

	cfg, err := ParseNginx("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseNginx error: %v", err)
	}

	if len(cfg) != 1 {
		t.Fatalf("expected 1 directive (comments stripped), got %d", len(cfg))
	}
	assertNginxDirective(t, cfg[0], "sendfile", []string{"on"}, false)
}

func TestParseNginx_Empty(t *testing.T) {
	cfg, err := ParseNginx("test", strings.NewReader(""))
	if err != nil {
		t.Fatalf("ParseNginx error: %v", err)
	}
	if len(cfg) != 0 {
		t.Errorf("expected empty config, got %d directives", len(cfg))
	}
}

func TestParseNginx_ErrorMissingSemicolon(t *testing.T) {
	input := `worker_processes 4`
	_, err := ParseNginx("test", strings.NewReader(input))
	if err == nil {
		t.Error("expected error for missing semicolon, got nil")
	}
}

func TestParseNginx_ErrorUnclosedBlock(t *testing.T) {
	input := `events {
    worker_connections 1024;`
	_, err := ParseNginx("test", strings.NewReader(input))
	if err == nil {
		t.Error("expected error for unclosed block, got nil")
	}
}

func TestParseNginx_ErrorUnexpectedCloseBrace(t *testing.T) {
	input := `}`
	_, err := ParseNginx("test", strings.NewReader(input))
	if err == nil {
		t.Error("expected error for unexpected '}', got nil")
	}
}

// ---- Lexer tests -------------------------------------------------------

func TestTokenize_BasicTokens(t *testing.T) {
	input := `label { directive arg1 }`
	tokens, err := tokenize("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	want := []string{"label", "{", "directive", "arg1", "}"}
	if len(tokens) != len(want) {
		t.Fatalf("expected %d tokens, got %d: %v", len(want), len(tokens), tokTexts(tokens))
	}
	for i, tok := range tokens {
		if tok.Text != want[i] {
			t.Errorf("token[%d]: expected %q, got %q", i, want[i], tok.Text)
		}
	}
}

func TestTokenize_Comments(t *testing.T) {
	input := "# full line comment\nword # inline comment"
	tokens, err := tokenize("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Text != "word" {
		t.Errorf("expected [word], got %v", tokTexts(tokens))
	}
}

func TestTokenize_QuotedStrings(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"hello world"`, "hello world"},
		{`'single quoted'`, "single quoted"},
		{`"escape\nnewline"`, "escape\nnewline"},
		{`"back\\slash"`, `back\slash`},
	}

	for _, tc := range tests {
		tokens, err := tokenize("test", strings.NewReader(tc.input))
		if err != nil {
			t.Errorf("input %q: tokenize error: %v", tc.input, err)
			continue
		}
		if len(tokens) != 1 || tokens[0].Text != tc.want {
			t.Errorf("input %q: expected [%q], got %v", tc.input, tc.want, tokTexts(tokens))
		}
	}
}

func TestTokenize_LineNumbers(t *testing.T) {
	input := "a\nb\nc"
	tokens, err := tokenize("test", strings.NewReader(input))
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	for i, tok := range tokens {
		wantLine := i + 1
		if tok.Line != wantLine {
			t.Errorf("token %q: expected line %d, got %d", tok.Text, wantLine, tok.Line)
		}
	}
}

func TestTokenize_UnterminatedString(t *testing.T) {
	_, err := tokenize("test", strings.NewReader(`"unterminated`))
	if err == nil {
		t.Error("expected error for unterminated string, got nil")
	}
}

// ---- helpers ------------------------------------------------------------------

func assertKeys(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("keys: expected %v, got %v", want, got)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("keys[%d]: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func assertSegment(t *testing.T, seg Segment, name string, args []string, blockLen int) {
	t.Helper()
	if seg.Name != name {
		t.Errorf("segment name: expected %q, got %q", name, seg.Name)
	}
	if len(seg.Args) != len(args) {
		t.Errorf("segment %q args: expected %v, got %v", name, args, seg.Args)
	} else {
		for i, a := range args {
			if seg.Args[i] != a {
				t.Errorf("segment %q args[%d]: expected %q, got %q", name, i, a, seg.Args[i])
			}
		}
	}
	if len(seg.Block) != blockLen {
		t.Errorf("segment %q block len: expected %d, got %d", name, blockLen, len(seg.Block))
	}
}

func assertNginxDirective(t *testing.T, d *Directive, name string, params []string, hasBody bool) {
	t.Helper()
	if d.Name != name {
		t.Errorf("directive name: expected %q, got %q", name, d.Name)
	}
	if len(d.Params) != len(params) {
		t.Errorf("directive %q params: expected %v, got %v", name, params, d.Params)
	} else {
		for i, p := range params {
			if d.Params[i] != p {
				t.Errorf("directive %q params[%d]: expected %q, got %q", name, i, p, d.Params[i])
			}
		}
	}
	if hasBody != (d.Body != nil) {
		t.Errorf("directive %q: hasBody expected %v, got body=%v", name, hasBody, d.Body)
	}
}

func findNginxDirective(cfg Config, name string) int {
	for i, d := range cfg {
		if d.Name == name {
			return i
		}
	}
	return -1
}

func segNames(segs []Segment) []string {
	names := make([]string, len(segs))
	for i, s := range segs {
		names[i] = s.Name
	}
	return names
}

func tokTexts(tokens []Token) []string {
	texts := make([]string, len(tokens))
	for i, t := range tokens {
		texts[i] = t.Text
	}
	return texts
}
