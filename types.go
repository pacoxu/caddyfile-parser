// Package caddyfileparser parses Caddyfile, DNS Corefile, and Nginx
// configuration file formats into structured Go types.
package caddyfileparser

// ServerBlock represents a labeled configuration block in Caddyfile or
// Corefile syntax. Each block has one or more keys (addresses or labels)
// followed by a body of directives enclosed in braces.
//
// Example:
//
//	.:53 {
//	    errors
//	    kubernetes cluster.local {
//	        pods insecure
//	    }
//	}
type ServerBlock struct {
	// Keys are the labels before the opening brace,
	// e.g. [".:53"] or ["localhost", "example.com"].
	Keys []string

	// Segments are the directives inside the block, in order.
	Segments []Segment
}

// Segment represents a single directive inside a ServerBlock. It holds the
// directive name, its arguments on the same line, and an optional nested
// sub-block.
//
// Example:
//
//	kubernetes cluster.local in-addr.arpa {
//	    pods insecure
//	    upstream
//	}
//
// produces Segment{Name: "kubernetes", Args: ["cluster.local", "in-addr.arpa"],
// Block: [...]}.
type Segment struct {
	// Name is the directive name (the first token on the directive line).
	Name string

	// Args are the arguments that follow the name on the same line.
	Args []string

	// Block holds sub-directives when the directive has a nested block.
	// It is nil when there is no sub-block.
	Block []Segment
}

// Directive represents a single directive in an Nginx configuration file.
// A directive is either a simple statement (ending with ";") or a block
// (containing nested directives inside "{ }").
//
// Example:
//
//	worker_processes 4;            → Directive{Name:"worker_processes", Params:["4"]}
//	events { worker_connections 1024; } → Directive{Name:"events", Body:[...]}
type Directive struct {
	// Name is the directive keyword.
	Name string

	// Params are the arguments that follow the name, before ";" or "{".
	Params []string

	// Body holds nested directives for block directives.
	// It is nil for simple (semicolon-terminated) directives.
	Body []*Directive
}

// Config is a parsed Nginx configuration file represented as a flat list of
// top-level directives (both simple directives and block directives).
type Config []*Directive
