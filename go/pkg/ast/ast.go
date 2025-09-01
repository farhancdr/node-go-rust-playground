package ast

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
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
	"String": "Str",
	"Int":    "Int",
	"Any":    "Any",
	"Error":  "Err",
	"Bool":   "Bool", // Added support for zap.Bool
}

func ZapToZero() {
	fileFlag := flag.String("file", "", "Go source file to process")
	dirFlag := flag.String("dir", "", "Directory to process recursively")
	inplace := flag.Bool("inplace", false, "Modify files in-place")
	flag.Parse()

	if *fileFlag == "" && *dirFlag == "" {
		fmt.Println("Please provide -file or -dir")
		os.Exit(1)
	}

	if *fileFlag != "" {
		processFile(*fileFlag, *inplace)
	} else {
		err := filepath.Walk(*dirFlag, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".go" {
				processFile(path, *inplace)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error walking directory: %v\n", err)
			os.Exit(1)
		}
	}
}

func processFile(path string, inplace bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Error parsing file %s: %v\n", path, err)
		return
	}

	modified := modifyAST(f)

	if modified {
		var buf bytes.Buffer
		cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 4}
		err = cfg.Fprint(&buf, fset, f)
		if err != nil {
			fmt.Printf("Error printing file %s: %v\n", path, err)
			return
		}

		output := buf.Bytes()
		if inplace {
			err = ioutil.WriteFile(path, output, 0644)
			if err != nil {
				fmt.Printf("Error writing file %s: %v\n", path, err)
			}
		} else {
			fmt.Println(buf.String())
		}
	}
}

func modifyAST(f *ast.File) bool {
	modified := false
	ast.Inspect(f, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			if processFunc(fd) {
				modified = true
			}
			return false
		}
		return true
	})

	if modified {
		if !isImportPresent(f, "github.com/rs/zerolog") {
			addImport(f, "github.com/rs/zerolog", "")
		}
	}

	if !usesZap(f) {
		removeImport(f, "go.uber.org/zap")
	}

	return modified
}

func processFunc(fd *ast.FuncDecl) bool {
	if fd.Body == nil {
		return false
	}

	if !hasZapLoggerCalls(fd.Body) {
		return false
	}

	if !hasLoggerDecl(fd.Body) {
		// todo
	}

	fd.Body = rewriteBlock(fd.Body)
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

func hasLoggerDecl(b *ast.BlockStmt) bool {
	if len(b.List) == 0 {
		return false
	}
	first, ok := b.List[0].(*ast.AssignStmt)
	if !ok {
		return false
	}
	if len(first.Lhs) != 1 || len(first.Rhs) != 1 {
		return false
	}
	lhs, ok := first.Lhs[0].(*ast.Ident)
	if !ok || lhs.Name != "logger" {
		return false
	}
	rhs, ok := first.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := rhs.Fun.(*ast.SelectorExpr)
	if !ok || !isUtilsGetLoggerFromContext(sel) {
		return false
	}
	if len(rhs.Args) != 1 {
		return false
	}
	argSel, ok := rhs.Args[0].(*ast.SelectorExpr)
	if !ok {
		return false
	}
	argX, ok := argSel.X.(*ast.Ident)
	if !ok || argX.Name != "r" || argSel.Sel.Name != "ctx" {
		return false
	}
	return true
}

func isUtilsLogger(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok || x.Name != "utils" {
		return false
	}
	return sel.Sel.Name == "Logger"
}

func isUtilsGetLoggerFromContext(sel *ast.SelectorExpr) bool {
	x, ok := sel.X.(*ast.Ident)
	if !ok || x.Name != "utils" {
		return false
	}
	return sel.Sel.Name == "GetLoggerFromContext"
}

func rewriteBlock(b *ast.BlockStmt) *ast.BlockStmt {
	for i := range b.List {
		b.List[i] = rewriteStmt(b.List[i])
	}
	return b
}

func rewriteStmt(s ast.Stmt) ast.Stmt {
	switch x := s.(type) {
	case *ast.BadStmt:
		return x
	case *ast.DeclStmt:
		return x
	case *ast.EmptyStmt:
		return x
	case *ast.LabeledStmt:
		x.Stmt = rewriteStmt(x.Stmt)
		return x
	case *ast.ExprStmt:
		x.X = rewriteExpr(x.X)
		return x
	case *ast.SendStmt:
		x.Chan = rewriteExpr(x.Chan)
		x.Value = rewriteExpr(x.Value)
		return x
	case *ast.IncDecStmt:
		x.X = rewriteExpr(x.X)
		return x
	case *ast.AssignStmt:
		for i := range x.Lhs {
			x.Lhs[i] = rewriteExpr(x.Lhs[i])
		}
		for i := range x.Rhs {
			x.Rhs[i] = rewriteExpr(x.Rhs[i])
		}
		return x
	case *ast.GoStmt:
		x.Call = rewriteExpr(x.Call).(*ast.CallExpr)
		return x
	case *ast.DeferStmt:
		x.Call = rewriteExpr(x.Call).(*ast.CallExpr)
		return x
	case *ast.ReturnStmt:
		for i := range x.Results {
			x.Results[i] = rewriteExpr(x.Results[i])
		}
		return x
	case *ast.BranchStmt:
		return x
	case *ast.BlockStmt:
		return rewriteBlock(x)
	case *ast.IfStmt:
		if x.Init != nil {
			x.Init = rewriteStmt(x.Init)
		}
		x.Cond = rewriteExpr(x.Cond)
		x.Body = rewriteBlock(x.Body)
		if x.Else != nil {
			x.Else = rewriteStmt(x.Else)
		}
		return x
	case *ast.CaseClause:
		for i := range x.List {
			x.List[i] = rewriteExpr(x.List[i])
		}
		for i := range x.Body {
			x.Body[i] = rewriteStmt(x.Body[i])
		}
		return x
	case *ast.SwitchStmt:
		if x.Init != nil {
			x.Init = rewriteStmt(x.Init)
		}
		if x.Tag != nil {
			x.Tag = rewriteExpr(x.Tag)
		}
		x.Body = rewriteBlock(x.Body)
		return x
	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			x.Init = rewriteStmt(x.Init)
		}
		x.Assign = rewriteStmt(x.Assign)
		x.Body = rewriteBlock(x.Body)
		return x
	case *ast.CommClause:
		if x.Comm != nil {
			x.Comm = rewriteStmt(x.Comm)
		}
		for i := range x.Body {
			x.Body[i] = rewriteStmt(x.Body[i])
		}
		return x
	case *ast.SelectStmt:
		x.Body = rewriteBlock(x.Body)
		return x
	case *ast.ForStmt:
		if x.Init != nil {
			x.Init = rewriteStmt(x.Init)
		}
		if x.Cond != nil {
			x.Cond = rewriteExpr(x.Cond)
		}
		if x.Post != nil {
			x.Post = rewriteStmt(x.Post)
		}
		x.Body = rewriteBlock(x.Body)
		return x
	case *ast.RangeStmt:
		if x.Key != nil {
			x.Key = rewriteExpr(x.Key)
		}
		if x.Value != nil {
			x.Value = rewriteExpr(x.Value)
		}
		x.X = rewriteExpr(x.X)
		x.Body = rewriteBlock(x.Body)
		return x
	default:
		fmt.Printf("Unhandled stmt type: %T\n", x)
		return x
	}
}

func rewriteExpr(e ast.Expr) ast.Expr {
	if e == nil {
		return nil
	}
	switch x := e.(type) {
	case *ast.BadExpr:
		return x
	case *ast.Ident:
		return x
	case *ast.BasicLit:
		return x
	case *ast.FuncLit:
		x.Body = rewriteBlock(x.Body)
		return x
	case *ast.CompositeLit:
		x.Type = rewriteExpr(x.Type)
		for i := range x.Elts {
			x.Elts[i] = rewriteExpr(x.Elts[i])
		}
		return x
	case *ast.ParenExpr:
		x.X = rewriteExpr(x.X)
		return x
	case *ast.SelectorExpr:
		x.X = rewriteExpr(x.X)
		return x
	case *ast.IndexExpr:
		x.X = rewriteExpr(x.X)
		x.Index = rewriteExpr(x.Index)
		return x
	case *ast.SliceExpr:
		x.X = rewriteExpr(x.X)
		if x.Low != nil {
			x.Low = rewriteExpr(x.Low)
		}
		if x.High != nil {
			x.High = rewriteExpr(x.High)
		}
		if x.Max != nil {
			x.Max = rewriteExpr(x.Max)
		}
		return x
	case *ast.TypeAssertExpr:
		x.X = rewriteExpr(x.X)
		x.Type = rewriteExpr(x.Type)
		return x
	case *ast.CallExpr:
		x.Fun = rewriteExpr(x.Fun)
		for i := range x.Args {
			x.Args[i] = rewriteExpr(x.Args[i])
		}
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
			if isUtilsLogger(sel.X) && logLevels[sel.Sel.Name] {
				return createZerologCall(sel.Sel.Name, x.Args)
			}
		}
		return x
	case *ast.StarExpr:
		x.X = rewriteExpr(x.X)
		return x
	case *ast.UnaryExpr:
		x.X = rewriteExpr(x.X)
		return x
	case *ast.BinaryExpr:
		x.X = rewriteExpr(x.X)
		x.Y = rewriteExpr(x.Y)
		return x
	case *ast.KeyValueExpr:
		x.Key = rewriteExpr(x.Key)
		x.Value = rewriteExpr(x.Value)
		return x
	case *ast.ArrayType:
		x.Len = rewriteExpr(x.Len)
		x.Elt = rewriteExpr(x.Elt)
		return x
	case *ast.StructType:
		return x
	case *ast.FuncType:
		return x
	case *ast.InterfaceType:
		return x
	case *ast.MapType:
		x.Key = rewriteExpr(x.Key)
		x.Value = rewriteExpr(x.Value)
		return x
	case *ast.ChanType:
		x.Value = rewriteExpr(x.Value)
		return x
	default:
		fmt.Printf("Unhandled expr type: %T\n", x)
		return x
	}
}

func createZerologCall(level string, args []ast.Expr) ast.Expr {
	if len(args) < 1 {
		panic("Invalid log call: no arguments")
	}
	msg := args[0]
	fields := args[1:]

	base := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("logger"),
			Sel: ast.NewIdent(level),
		},
	}
	curr := base

	for _, field := range fields {
		fcall, ok := field.(*ast.CallExpr)
		if !ok {
			panic("Field is not a call expression")
		}
		fsel, ok := fcall.Fun.(*ast.SelectorExpr)
		if !ok {
			panic("Field fun is not selector")
		}
		fx, ok := fsel.X.(*ast.Ident)
		if !ok || fx.Name != "zap" {
			panic("Field not from zap")
		}
		zapType := fsel.Sel.Name
		zeroType, ok := zapToZero[zapType]
		if !ok {
			fmt.Printf("Unknown zap field type: %s\n", zapType)
			return base // Skip unsupported field types gracefully
		}
		newCall := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   curr,
				Sel: ast.NewIdent(zeroType),
			},
			Args: fcall.Args,
		}
		curr = newCall
	}

	msgCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   curr,
			Sel: ast.NewIdent("Msg"),
		},
		Args: []ast.Expr{msg},
	}
	return msgCall
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
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + path + `"`,
		},
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
		importDecl = &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: []ast.Spec{},
		}
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
