package ast2

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
)

var logLevels = map[string]bool{
	"Debug": true,
	"Info":  true,
	"Warn":  true,
	"Error": true,
	"Panic": true,
	"Fatal": true,
}

var zapToZero = map[string]string{
	"String":   "Str",
	"Int":      "Int",
	"Int64":    "Int64",
	"Uint":     "Uint",
	"Uint64":   "Uint64",
	"Bool":     "Bool",
	"Float64":  "Float64",
	"Duration": "Dur",
	"Time":     "Time",
	"Any":      "Interface",
	"Error":    "Err",
}

func ZapToZero2() {
	fileFlag := flag.String("file", "", "Go source file to process")
	dirFlag := flag.String("dir", "", "Directory to process recursively")
	inplace := flag.Bool("inplace", false, "Modify files in-place")
	flag.Parse()

	fmt.Printf("ZapToZero2 version")

	if *fileFlag == "" && *dirFlag == "" {
		fmt.Println("Please provide -file or -dir")
		os.Exit(1)
	}

	if *fileFlag != "" {
		if err := processFile(*fileFlag, *inplace); err != nil {
			fmt.Printf("Error processing file %s: %v\n", *fileFlag, err)
			os.Exit(1)
		}
		return
	}

	err := filepath.Walk(*dirFlag, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".go" {
			if err := processFile(path, *inplace); err != nil {
				fmt.Printf("Error processing file %s: %v\n", path, err)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}
}

func processFile(path string, inplace bool) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	// Find receiver names for methods
	receivers := make(map[*ast.BlockStmt]string)
	ast.Inspect(f, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Recv != nil && len(fd.Recv.List) > 0 && len(fd.Recv.List[0].Names) > 0 {
			receivers[fd.Body] = fd.Recv.List[0].Names[0].Name
		}
		return true
	})

	modified := modifyAST(f, receivers)

	if !modified {
		return nil
	}

	var buf bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 4}
	if err := cfg.Fprint(&buf, fset, f); err != nil {
		return fmt.Errorf("printing file: %w", err)
	}

	if inplace {
		if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
	} else {
		fmt.Println(buf.String())
	}
	return nil
}

func modifyAST(f *ast.File, receivers map[*ast.BlockStmt]string) bool {
	modified := false
	ast.Inspect(f, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Body != nil {
			if processFunc(fd, receivers) {
				modified = true
			}
			return false
		}
		return true
	})

	if modified {
		if !isImportPresent(f, "github.com/pkg/errors") {
			addImport(f, "github.com/pkg/errors", "")
		}
		if !isImportPresent(f, "github.com/rs/zerolog") {
			addImport(f, "github.com/rs/zerolog", "")
		}
		if !usesZap(f) {
			removeImport(f, "go.uber.org/zap")
		}
	}
	return modified
}

func processFunc(fd *ast.FuncDecl, receivers map[*ast.BlockStmt]string) bool {
	if fd.Body == nil {
		return false
	}
	if !hasZapLoggerCalls(fd.Body) {
		return false
	}
	recv, ok := receivers[fd.Body]
	if !ok {
		return false // Skip if no receiver
	}
	fd.Body = rewriteBlock(fd.Body, recv)
	return true
}

func hasZapLoggerCalls(b *ast.BlockStmt) bool {
	found := false
	ast.Inspect(b, func(n ast.Node) bool {
		if found {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if isUtilsLogger(sel.X) && logLevels[sel.Sel.Name] {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

func isUtilsLogger(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "utils" && sel.Sel.Name == "Logger"
}

func rewriteBlock(b *ast.BlockStmt, recv string) *ast.BlockStmt {
	for i := range b.List {
		b.List[i] = rewriteStmt(b.List[i], recv)
	}
	return b
}

func rewriteStmt(s ast.Stmt, recv string) ast.Stmt {
	switch x := s.(type) {
	case *ast.ExprStmt:
		x.X = rewriteExpr(x.X, recv)
	case *ast.AssignStmt:
		for i := range x.Lhs {
			x.Lhs[i] = rewriteExpr(x.Lhs[i], recv)
		}
		for i := range x.Rhs {
			x.Rhs[i] = rewriteExpr(x.Rhs[i], recv)
		}
	case *ast.IfStmt:
		if x.Init != nil {
			x.Init = rewriteStmt(x.Init, recv)
		}
		x.Cond = rewriteExpr(x.Cond, recv)
		x.Body = rewriteBlock(x.Body, recv)
		if x.Else != nil {
			x.Else = rewriteStmt(x.Else, recv)
		}
	case *ast.BlockStmt:
		rewriteBlock(x, recv)
	case *ast.ForStmt:
		if x.Init != nil {
			x.Init = rewriteStmt(x.Init, recv)
		}
		if x.Cond != nil {
			x.Cond = rewriteExpr(x.Cond, recv)
		}
		if x.Post != nil {
			x.Post = rewriteStmt(x.Post, recv)
		}
		x.Body = rewriteBlock(x.Body, recv)
	case *ast.RangeStmt:
		if x.Key != nil {
			x.Key = rewriteExpr(x.Key, recv)
		}
		if x.Value != nil {
			x.Value = rewriteExpr(x.Value, recv)
		}
		x.X = rewriteExpr(x.X, recv)
		x.Body = rewriteBlock(x.Body, recv)
	case *ast.SwitchStmt:
		if x.Init != nil {
			x.Init = rewriteStmt(x.Init, recv)
		}
		if x.Tag != nil {
			x.Tag = rewriteExpr(x.Tag, recv)
		}
		x.Body = rewriteBlock(x.Body, recv)
	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			x.Init = rewriteStmt(x.Init, recv)
		}
		x.Assign = rewriteStmt(x.Assign, recv)
		x.Body = rewriteBlock(x.Body, recv)
	case *ast.DeferStmt:
		x.Call = rewriteExpr(x.Call, recv).(*ast.CallExpr)
	case *ast.GoStmt:
		x.Call = rewriteExpr(x.Call, recv).(*ast.CallExpr)
	case *ast.ReturnStmt:
		for i := range x.Results {
			x.Results[i] = rewriteExpr(x.Results[i], recv)
		}
	case *ast.LabeledStmt:
		x.Stmt = rewriteStmt(x.Stmt, recv)
	case *ast.SendStmt:
		x.Chan = rewriteExpr(x.Chan, recv)
		x.Value = rewriteExpr(x.Value, recv)
	case *ast.IncDecStmt:
		x.X = rewriteExpr(x.X, recv)
	case *ast.CommClause:
		if x.Comm != nil {
			x.Comm = rewriteStmt(x.Comm, recv)
		}
		for i := range x.Body {
			x.Body[i] = rewriteStmt(x.Body[i], recv)
		}
	case *ast.SelectStmt:
		x.Body = rewriteBlock(x.Body, recv)
	case *ast.CaseClause:
		for i := range x.List {
			x.List[i] = rewriteExpr(x.List[i], recv)
		}
		for i := range x.Body {
			x.Body[i] = rewriteStmt(x.Body[i], recv)
		}
	}
	return s
}

func rewriteExpr(e ast.Expr, recv string) ast.Expr {
	if e == nil {
		return nil
	}
	switch x := e.(type) {
	case *ast.CallExpr:
		x.Fun = rewriteExpr(x.Fun, recv)
		for i := range x.Args {
			x.Args[i] = rewriteExpr(x.Args[i], recv)
		}
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
			if isUtilsLogger(sel.X) && logLevels[sel.Sel.Name] {
				return createZerologCall(sel.Sel.Name, x.Args, recv)
			}
		}
	case *ast.ParenExpr:
		x.X = rewriteExpr(x.X, recv)
	case *ast.SelectorExpr:
		x.X = rewriteExpr(x.X, recv)
	case *ast.IndexExpr:
		x.X = rewriteExpr(x.X, recv)
		x.Index = rewriteExpr(x.Index, recv)
	case *ast.SliceExpr:
		x.X = rewriteExpr(x.X, recv)
		if x.Low != nil {
			x.Low = rewriteExpr(x.Low, recv)
		}
		if x.High != nil {
			x.High = rewriteExpr(x.High, recv)
		}
		if x.Max != nil {
			x.Max = rewriteExpr(x.Max, recv)
		}
	case *ast.TypeAssertExpr:
		x.X = rewriteExpr(x.X, recv)
		x.Type = rewriteExpr(x.Type, recv)
	case *ast.FuncLit:
		x.Body = rewriteBlock(x.Body, recv)
	case *ast.CompositeLit:
		x.Type = rewriteExpr(x.Type, recv)
		for i := range x.Elts {
			x.Elts[i] = rewriteExpr(x.Elts[i], recv)
		}
	case *ast.StarExpr:
		x.X = rewriteExpr(x.X, recv)
	case *ast.UnaryExpr:
		x.X = rewriteExpr(x.X, recv)
	case *ast.BinaryExpr:
		x.X = rewriteExpr(x.X, recv)
		x.Y = rewriteExpr(x.Y, recv)
	case *ast.KeyValueExpr:
		x.Key = rewriteExpr(x.Key, recv)
		x.Value = rewriteExpr(x.Value, recv)
	case *ast.ArrayType:
		x.Len = rewriteExpr(x.Len, recv)
		x.Elt = rewriteExpr(x.Elt, recv)
	case *ast.MapType:
		x.Key = rewriteExpr(x.Key, recv)
		x.Value = rewriteExpr(x.Value, recv)
	case *ast.ChanType:
		x.Value = rewriteExpr(x.Value, recv)
	}
	return e
}

func createZerologCall(level string, args []ast.Expr, recv string) ast.Expr {
	if len(args) < 1 {
		return args[0] // Skip invalid calls
	}

	msg := args[0]
	fields := args[1:]

	// Check if msg is err.Error()
	var errExpr ast.Expr
	isErrMsg := false
	if mcall, ok := msg.(*ast.CallExpr); ok {
		if msel, ok := mcall.Fun.(*ast.SelectorExpr); ok {
			if msel.Sel.Name == "Error" && len(mcall.Args) == 0 {
				errExpr = msel.X
				isErrMsg = true
			}
		}
	}

	// Start chain: r.logger.Level()
	base := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.SelectorExpr{X: ast.NewIdent(recv), Sel: ast.NewIdent("logger")},
			Sel: ast.NewIdent(level),
		},
	}
	curr := base

	// Add fields
	for _, field := range fields {
		fcall, ok := field.(*ast.CallExpr)
		if !ok {
			continue
		}
		fsel, ok := fcall.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		fx, ok := fsel.X.(*ast.Ident)
		if !ok || fx.Name != "zap" {
			continue
		}
		zapType := fsel.Sel.Name
		zeroType, ok := zapToZero[zapType]
		if !ok {
			fmt.Printf("Skipping unknown zap field type: %s\n", zapType)
			continue
		}

		var args []ast.Expr
		if zapType == "Error" {
			if len(fcall.Args) != 1 {
				continue
			}
			args = []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("errors"),
						Sel: ast.NewIdent("Wrap"),
					},
					Args: []ast.Expr{
						fcall.Args[0],
						&ast.BasicLit{Kind: token.STRING, Value: `"from error"`},
					},
				},
			}
		} else {
			args = fcall.Args
		}

		newCall := &ast.CallExpr{
			Fun:  &ast.SelectorExpr{X: curr, Sel: ast.NewIdent(zeroType)},
			Args: args,
		}
		curr = newCall
	}

	// If msg was err.Error(), add .Err() and set msg to ""
	if isErrMsg {
		newCall := &ast.CallExpr{
			Fun: &ast.SelectorExpr{X: curr, Sel: ast.NewIdent("Err")},
			Args: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("errors"),
						Sel: ast.NewIdent("Wrap"),
					},
					Args: []ast.Expr{
						errExpr,
						&ast.BasicLit{Kind: token.STRING, Value: `"from error"`},
					},
				},
			},
		}
		curr = newCall
		msg = &ast.BasicLit{Kind: token.STRING, Value: `""`}
	}

	// Add .Msg(msg)
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: curr, Sel: ast.NewIdent("Msg")},
		Args: []ast.Expr{msg},
	}
}

func isImportPresent(f *ast.File, path string) bool {
	for _, imp := range f.Imports {
		if imp.Path.Value == `"`+path+`"` {
			return true
		}
	}
	return false
}

func addImport(f *ast.File, path, name string) {
	imp := &ast.ImportSpec{
		Path: &ast.BasicLit{Kind: token.STRING, Value: `"` + path + `"`},
	}
	if name != "" {
		imp.Name = ast.NewIdent(name)
	}

	var importDecl *ast.GenDecl
	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			importDecl = gd
			break
		}
	}
	if importDecl == nil {
		importDecl = &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{}}
		f.Decls = append([]ast.Decl{importDecl}, f.Decls...)
	}
	importDecl.Specs = append(importDecl.Specs, imp)
}

func removeImport(f *ast.File, path string) {
	newDecls := []ast.Decl{}
	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			newSpecs := []ast.Spec{}
			for _, spec := range gd.Specs {
				if imp, ok := spec.(*ast.ImportSpec); ok && imp.Path.Value != `"`+path+`"` {
					newSpecs = append(newSpecs, spec)
				}
			}
			if len(newSpecs) > 0 {
				gd.Specs = newSpecs
				newDecls = append(newDecls, gd)
			}
		} else {
			newDecls = append(newDecls, decl)
		}
	}
	f.Decls = newDecls
}

func usesZap(f *ast.File) bool {
	used := false
	ast.Inspect(f, func(n ast.Node) bool {
		if used {
			return false
		}
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "zap" {
				used = true
				return false
			}
		}
		return true
	})
	return used
}
