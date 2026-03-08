package scanner

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// Symbol represents a named declaration in source code.
type Symbol struct {
	Name       string
	Kind       string // "function", "class", "interface", "method", "struct"
	StartLine  int
	EndLine    int
	Docstring  string
	IsExported bool
}

// ImportDecl represents a single import statement.
type ImportDecl struct {
	Source string   // the module/package path
	Names  []string // imported identifiers (empty for wildcard/default)
	IsWild bool     // import * or wildcard
}

// ParsedFile contains all extracted information from a single source file.
type ParsedFile struct {
	Path       string
	Language   string
	Functions  []Symbol
	Classes    []Symbol
	Interfaces []Symbol
	Imports    []ImportDecl
	Exports    []string
	Comments   []string
	LineCount  int
}

// languageGrammars maps language names to their Tree-sitter grammar.
var languageGrammars = map[string]*sitter.Language{
	"TypeScript": typescript.GetLanguage(),
	"JavaScript": javascript.GetLanguage(),
	"Python":     python.GetLanguage(),
	"Go":         golang.GetLanguage(),
	"Rust":       rust.GetLanguage(),
	"Ruby":       ruby.GetLanguage(),
	"Java":       java.GetLanguage(),
}

// Parse extracts symbols, imports, and exports from a source file using Tree-sitter.
func Parse(ctx context.Context, entry FileEntry, content []byte) (*ParsedFile, error) {
	lang := languageGrammars[entry.Language]
	if lang == nil {
		return nil, fmt.Errorf("unsupported language for parsing: %s", entry.Language)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", entry.RelPath, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	lineCount := strings.Count(string(content), "\n") + 1

	pf := &ParsedFile{
		Path:      entry.RelPath,
		Language:  entry.Language,
		LineCount: lineCount,
	}

	switch entry.Language {
	case "TypeScript", "JavaScript":
		extractTS(root, content, pf)
	case "Python":
		extractPython(root, content, pf)
	case "Go":
		extractGo(root, content, pf)
	case "Rust":
		extractRust(root, content, pf)
	case "Ruby":
		extractRuby(root, content, pf)
	case "Java":
		extractJava(root, content, pf)
	}

	return pf, nil
}

// SupportedForParsing returns true if the language has a Tree-sitter grammar.
func SupportedForParsing(language string) bool {
	_, ok := languageGrammars[language]
	return ok
}

// --- TypeScript / JavaScript ---

func extractTS(root *sitter.Node, src []byte, pf *ParsedFile) {
	walkNodes(root, func(node *sitter.Node) {
		switch node.Type() {
		case "function_declaration":
			if name := childByField(node, "name", src); name != "" {
				pf.Functions = append(pf.Functions, Symbol{
					Name:       name,
					Kind:       "function",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					Docstring:  precedingComment(node, src),
					IsExported: isExportedTS(node),
				})
			}

		case "class_declaration":
			if name := childByField(node, "name", src); name != "" {
				pf.Classes = append(pf.Classes, Symbol{
					Name:       name,
					Kind:       "class",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					Docstring:  precedingComment(node, src),
					IsExported: isExportedTS(node),
				})
			}

		case "interface_declaration":
			if name := childByField(node, "name", src); name != "" {
				pf.Interfaces = append(pf.Interfaces, Symbol{
					Name:       name,
					Kind:       "interface",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					Docstring:  precedingComment(node, src),
					IsExported: isExportedTS(node),
				})
			}

		case "method_definition":
			if name := childByField(node, "name", src); name != "" {
				pf.Functions = append(pf.Functions, Symbol{
					Name:      name,
					Kind:      "method",
					StartLine: int(node.StartPoint().Row) + 1,
					EndLine:   int(node.EndPoint().Row) + 1,
					Docstring: precedingComment(node, src),
				})
			}

		case "arrow_function":
			// Only capture if assigned to a variable
			parent := node.Parent()
			if parent != nil && parent.Type() == "variable_declarator" {
				if name := childByField(parent, "name", src); name != "" {
					pf.Functions = append(pf.Functions, Symbol{
						Name:       name,
						Kind:       "function",
						StartLine:  int(node.StartPoint().Row) + 1,
						EndLine:    int(node.EndPoint().Row) + 1,
						IsExported: isExportedTS(parent),
					})
				}
			}

		case "import_statement":
			imp := extractTSImport(node, src)
			if imp != nil {
				pf.Imports = append(pf.Imports, *imp)
			}

		case "export_statement":
			extractTSExport(node, src, pf)
		}
	})
}

func extractTSImport(node *sitter.Node, src []byte) *ImportDecl {
	imp := &ImportDecl{}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "string", "string_fragment":
			imp.Source = unquote(nodeText(child, src))
		case "import_clause":
			for j := 0; j < int(child.ChildCount()); j++ {
				clause := child.Child(j)
				switch clause.Type() {
				case "identifier":
					imp.Names = append(imp.Names, nodeText(clause, src))
				case "named_imports":
					for k := 0; k < int(clause.ChildCount()); k++ {
						spec := clause.Child(k)
						if spec.Type() == "import_specifier" {
							if name := childByField(spec, "name", src); name != "" {
								imp.Names = append(imp.Names, name)
							}
						}
					}
				case "namespace_import":
					imp.IsWild = true
				}
			}
		}
	}

	// If source wasn't found in direct children, search deeper
	if imp.Source == "" {
		walkNodes(node, func(n *sitter.Node) {
			if n.Type() == "string_fragment" && imp.Source == "" {
				imp.Source = nodeText(n, src)
			}
		})
	}

	if imp.Source == "" {
		return nil
	}
	return imp
}

func extractTSExport(node *sitter.Node, src []byte, pf *ParsedFile) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "function_declaration":
			if name := childByField(child, "name", src); name != "" {
				pf.Exports = append(pf.Exports, name)
			}
		case "class_declaration":
			if name := childByField(child, "name", src); name != "" {
				pf.Exports = append(pf.Exports, name)
			}
		case "interface_declaration":
			if name := childByField(child, "name", src); name != "" {
				pf.Exports = append(pf.Exports, name)
			}
		case "lexical_declaration":
			walkNodes(child, func(n *sitter.Node) {
				if n.Type() == "variable_declarator" {
					if name := childByField(n, "name", src); name != "" {
						pf.Exports = append(pf.Exports, name)
					}
				}
			})
		case "identifier":
			pf.Exports = append(pf.Exports, nodeText(child, src))
		}
	}
}

func isExportedTS(node *sitter.Node) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}
	// export function/class/const ...
	if parent.Type() == "export_statement" {
		return true
	}
	// const x = ... inside export
	if parent.Type() == "lexical_declaration" && parent.Parent() != nil && parent.Parent().Type() == "export_statement" {
		return true
	}
	return false
}

// --- Python ---

func extractPython(root *sitter.Node, src []byte, pf *ParsedFile) {
	walkNodes(root, func(node *sitter.Node) {
		switch node.Type() {
		case "function_definition":
			if name := childByField(node, "name", src); name != "" {
				pf.Functions = append(pf.Functions, Symbol{
					Name:       name,
					Kind:       "function",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					Docstring:  extractPythonDocstring(node, src),
					IsExported: !strings.HasPrefix(name, "_"),
				})
			}

		case "class_definition":
			if name := childByField(node, "name", src); name != "" {
				pf.Classes = append(pf.Classes, Symbol{
					Name:       name,
					Kind:       "class",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					Docstring:  extractPythonDocstring(node, src),
					IsExported: !strings.HasPrefix(name, "_"),
				})
			}

		case "import_statement":
			imp := extractPythonImport(node, src)
			if imp != nil {
				pf.Imports = append(pf.Imports, *imp)
			}

		case "import_from_statement":
			imp := extractPythonFromImport(node, src)
			if imp != nil {
				pf.Imports = append(pf.Imports, *imp)
			}
		}
	})

	// Python exports: all public names (no leading underscore) at module level
	for _, fn := range pf.Functions {
		if fn.IsExported {
			pf.Exports = append(pf.Exports, fn.Name)
		}
	}
	for _, cls := range pf.Classes {
		if cls.IsExported {
			pf.Exports = append(pf.Exports, cls.Name)
		}
	}
}

func extractPythonDocstring(node *sitter.Node, src []byte) string {
	// Look for string expression as first child of the body
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "block" && child.ChildCount() > 0 {
			first := child.Child(0)
			if first.Type() == "expression_statement" && first.ChildCount() > 0 {
				expr := first.Child(0)
				if expr.Type() == "string" {
					return strings.Trim(nodeText(expr, src), "\"'")
				}
			}
		}
	}
	return ""
}

func extractPythonImport(node *sitter.Node, src []byte) *ImportDecl {
	imp := &ImportDecl{}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "dotted_name" {
			imp.Source = nodeText(child, src)
			imp.Names = append(imp.Names, nodeText(child, src))
		}
	}
	if imp.Source == "" {
		return nil
	}
	return imp
}

func extractPythonFromImport(node *sitter.Node, src []byte) *ImportDecl {
	imp := &ImportDecl{}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "dotted_name", "relative_import":
			if imp.Source == "" {
				imp.Source = nodeText(child, src)
			}
		case "import_prefix":
			imp.Source = nodeText(child, src)
		case "wildcard_import":
			imp.IsWild = true
		}
	}

	// Extract individual imported names
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "dotted_name" && nodeText(child, src) != imp.Source {
			imp.Names = append(imp.Names, nodeText(child, src))
		}
	}

	if imp.Source == "" {
		return nil
	}
	return imp
}

// --- Go ---

func extractGo(root *sitter.Node, src []byte, pf *ParsedFile) {
	walkNodes(root, func(node *sitter.Node) {
		switch node.Type() {
		case "function_declaration":
			if name := childByField(node, "name", src); name != "" {
				pf.Functions = append(pf.Functions, Symbol{
					Name:       name,
					Kind:       "function",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					Docstring:  precedingComment(node, src),
					IsExported: isUpperFirst(name),
				})
			}

		case "method_declaration":
			if name := childByField(node, "name", src); name != "" {
				pf.Functions = append(pf.Functions, Symbol{
					Name:       name,
					Kind:       "method",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					Docstring:  precedingComment(node, src),
					IsExported: isUpperFirst(name),
				})
			}

		case "type_declaration":
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				switch child.Type() {
				case "type_spec":
					name := childByField(child, "name", src)
					if name == "" {
						continue
					}
					// Determine if struct or interface
					typeNode := childByField(child, "type", src)
					sym := Symbol{
						Name:       name,
						StartLine:  int(child.StartPoint().Row) + 1,
						EndLine:    int(child.EndPoint().Row) + 1,
						Docstring:  precedingComment(node, src),
						IsExported: isUpperFirst(name),
					}
					switch typeNode {
					case "struct_type":
						sym.Kind = "struct"
						pf.Classes = append(pf.Classes, sym)
					case "interface_type":
						sym.Kind = "interface"
						pf.Interfaces = append(pf.Interfaces, sym)
					default:
						sym.Kind = "type"
						pf.Classes = append(pf.Classes, sym)
					}
				}
			}

		case "import_declaration":
			extractGoImports(node, src, pf)

		case "package_clause":
			// not needed for imports
		}
	})

	// Go exports: uppercase names
	for _, fn := range pf.Functions {
		if fn.IsExported {
			pf.Exports = append(pf.Exports, fn.Name)
		}
	}
	for _, cls := range pf.Classes {
		if cls.IsExported {
			pf.Exports = append(pf.Exports, cls.Name)
		}
	}
	for _, iface := range pf.Interfaces {
		if iface.IsExported {
			pf.Exports = append(pf.Exports, iface.Name)
		}
	}
}

func extractGoImports(node *sitter.Node, src []byte, pf *ParsedFile) {
	walkNodes(node, func(n *sitter.Node) {
		if n.Type() == "import_spec" {
			path := ""
			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				if child.Type() == "interpreted_string_literal" {
					path = unquote(nodeText(child, src))
				}
			}
			if path != "" {
				pf.Imports = append(pf.Imports, ImportDecl{
					Source: path,
				})
			}
		}
	})
}

// --- Rust ---

func extractRust(root *sitter.Node, src []byte, pf *ParsedFile) {
	walkNodes(root, func(node *sitter.Node) {
		switch node.Type() {
		case "function_item":
			if name := childByField(node, "name", src); name != "" {
				pf.Functions = append(pf.Functions, Symbol{
					Name:       name,
					Kind:       "function",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					IsExported: hasPubModifier(node),
				})
			}

		case "struct_item":
			if name := childByField(node, "name", src); name != "" {
				pf.Classes = append(pf.Classes, Symbol{
					Name:       name,
					Kind:       "struct",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					IsExported: hasPubModifier(node),
				})
			}

		case "trait_item":
			if name := childByField(node, "name", src); name != "" {
				pf.Interfaces = append(pf.Interfaces, Symbol{
					Name:       name,
					Kind:       "trait",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					IsExported: hasPubModifier(node),
				})
			}

		case "impl_item":
			// Extract methods inside impl blocks
			if name := childByField(node, "type", src); name != "" {
				for i := 0; i < int(node.ChildCount()); i++ {
					child := node.Child(i)
					if child.Type() == "declaration_list" {
						walkNodes(child, func(m *sitter.Node) {
							if m.Type() == "function_item" {
								if mname := childByField(m, "name", src); mname != "" {
									pf.Functions = append(pf.Functions, Symbol{
										Name:       mname,
										Kind:       "method",
										StartLine:  int(m.StartPoint().Row) + 1,
										EndLine:    int(m.EndPoint().Row) + 1,
										IsExported: hasPubModifier(m),
									})
								}
							}
						})
					}
				}
			}

		case "use_declaration":
			imp := extractRustUse(node, src)
			if imp != nil {
				pf.Imports = append(pf.Imports, *imp)
			}
		}
	})

	for _, fn := range pf.Functions {
		if fn.IsExported {
			pf.Exports = append(pf.Exports, fn.Name)
		}
	}
	for _, cls := range pf.Classes {
		if cls.IsExported {
			pf.Exports = append(pf.Exports, cls.Name)
		}
	}
}

func hasPubModifier(node *sitter.Node) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == "visibility_modifier" {
			return true
		}
	}
	return false
}

func extractRustUse(node *sitter.Node, src []byte) *ImportDecl {
	text := nodeText(node, src)
	// Simplified: extract the use path
	text = strings.TrimPrefix(text, "use ")
	text = strings.TrimSuffix(text, ";")
	if text == "" {
		return nil
	}
	return &ImportDecl{Source: text}
}

// --- Ruby ---

func extractRuby(root *sitter.Node, src []byte, pf *ParsedFile) {
	walkNodes(root, func(node *sitter.Node) {
		switch node.Type() {
		case "method":
			if name := childByField(node, "name", src); name != "" {
				pf.Functions = append(pf.Functions, Symbol{
					Name:       name,
					Kind:       "method",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					IsExported: !strings.HasPrefix(name, "_"),
				})
			}

		case "class":
			if name := childByField(node, "name", src); name != "" {
				pf.Classes = append(pf.Classes, Symbol{
					Name:      name,
					Kind:      "class",
					StartLine: int(node.StartPoint().Row) + 1,
					EndLine:   int(node.EndPoint().Row) + 1,
				})
			}

		case "module":
			if name := childByField(node, "name", src); name != "" {
				pf.Classes = append(pf.Classes, Symbol{
					Name:      name,
					Kind:      "module",
					StartLine: int(node.StartPoint().Row) + 1,
					EndLine:   int(node.EndPoint().Row) + 1,
				})
			}

		case "call":
			text := nodeText(node, src)
			if strings.HasPrefix(text, "require ") || strings.HasPrefix(text, "require_relative ") {
				parts := strings.SplitN(text, " ", 2)
				if len(parts) == 2 {
					source := strings.Trim(parts[1], "\"'()")
					pf.Imports = append(pf.Imports, ImportDecl{Source: source})
				}
			}
		}
	})
}

// --- Java ---

func extractJava(root *sitter.Node, src []byte, pf *ParsedFile) {
	walkNodes(root, func(node *sitter.Node) {
		switch node.Type() {
		case "method_declaration":
			if name := childByField(node, "name", src); name != "" {
				pf.Functions = append(pf.Functions, Symbol{
					Name:       name,
					Kind:       "method",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					IsExported: hasJavaModifier(node, src, "public"),
				})
			}

		case "class_declaration":
			if name := childByField(node, "name", src); name != "" {
				pf.Classes = append(pf.Classes, Symbol{
					Name:       name,
					Kind:       "class",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					IsExported: hasJavaModifier(node, src, "public"),
				})
			}

		case "interface_declaration":
			if name := childByField(node, "name", src); name != "" {
				pf.Interfaces = append(pf.Interfaces, Symbol{
					Name:       name,
					Kind:       "interface",
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					IsExported: hasJavaModifier(node, src, "public"),
				})
			}

		case "import_declaration":
			text := nodeText(node, src)
			text = strings.TrimPrefix(text, "import ")
			text = strings.TrimPrefix(text, "static ")
			text = strings.TrimSuffix(text, ";")
			if text != "" {
				pf.Imports = append(pf.Imports, ImportDecl{
					Source: strings.TrimSpace(text),
					IsWild: strings.HasSuffix(text, ".*"),
				})
			}
		}
	})

	for _, fn := range pf.Functions {
		if fn.IsExported {
			pf.Exports = append(pf.Exports, fn.Name)
		}
	}
	for _, cls := range pf.Classes {
		if cls.IsExported {
			pf.Exports = append(pf.Exports, cls.Name)
		}
	}
}

func hasJavaModifier(node *sitter.Node, src []byte, modifier string) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "modifiers" {
			return strings.Contains(nodeText(child, src), modifier)
		}
	}
	return false
}

// --- Helper functions ---

func walkNodes(node *sitter.Node, fn func(n *sitter.Node)) {
	fn(node)
	for i := 0; i < int(node.ChildCount()); i++ {
		walkNodes(node.Child(i), fn)
	}
}

func nodeText(node *sitter.Node, src []byte) string {
	return string(src[node.StartByte():node.EndByte()])
}

func childByField(node *sitter.Node, field string, src []byte) string {
	child := node.ChildByFieldName(field)
	if child == nil {
		return ""
	}
	return nodeText(child, src)
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func precedingComment(node *sitter.Node, src []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}
	if prev.Type() == "comment" || prev.Type() == "line_comment" || prev.Type() == "block_comment" {
		text := nodeText(prev, src)
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		return strings.TrimSpace(text)
	}
	return ""
}

func isUpperFirst(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= 'A' && s[0] <= 'Z'
}
