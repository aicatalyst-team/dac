package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
)

// makeRequireFunc creates a CommonJS require() function for the goja runtime.
// It resolves paths relative to baseDir, transpiles .tsx files, and caches modules.
func makeRequireFunc(vm *goja.Runtime, baseDir string, cache map[string]goja.Value, cfg *tsxConfig, hFunc func(goja.FunctionCall) goja.Value) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		modPath := call.Argument(0).String()

		// Resolve the path relative to baseDir.
		resolved := resolveModulePath(baseDir, modPath)

		// Check cache.
		if cached, ok := cache[resolved]; ok {
			return cached
		}

		// Read the file.
		source, err := os.ReadFile(resolved)
		if err != nil {
			panic(vm.ToValue(fmt.Sprintf("require: cannot find module %q: %v", modPath, err)))
		}

		ext := filepath.Ext(resolved)
		var code string

		switch ext {
		case ".json":
			// JSON modules: parse and return.
			var v interface{}
			if err := json.Unmarshal(source, &v); err != nil {
				panic(vm.ToValue(fmt.Sprintf("require: invalid JSON in %q: %v", modPath, err)))
			}
			val := vm.ToValue(v)
			cache[resolved] = val
			return val

		case ".tsx", ".ts", ".jsx":
			// Transpile TSX/TS/JSX through esbuild.
			code, err = transpileTSX(string(source))
			if err != nil {
				panic(vm.ToValue(fmt.Sprintf("require: transpile error in %q: %v", modPath, err)))
			}

		default:
			// Plain JS — use as-is.
			code = string(source)
		}

		// Create module sandbox.
		modExports := vm.NewObject()
		mod := vm.NewObject()
		_ = mod.Set("exports", modExports)

		// Create a require function scoped to the module's directory.
		modDir := filepath.Dir(resolved)
		modRequire := makeRequireFunc(vm, modDir, cache, cfg, hFunc)

		wrapped := fmt.Sprintf(
			"(function(exports, module, require, h) {\n%s\n})",
			code,
		)

		compiled, err := goja.Compile(resolved, wrapped, false)
		if err != nil {
			panic(vm.ToValue(fmt.Sprintf("require: compile error in %q: %v", modPath, err)))
		}

		fnVal, err := vm.RunProgram(compiled)
		if err != nil {
			panic(vm.ToValue(fmt.Sprintf("require: eval error in %q: %v", modPath, err)))
		}

		fn, ok := goja.AssertFunction(fnVal)
		if !ok {
			panic(vm.ToValue(fmt.Sprintf("require: invalid module wrapper in %q", modPath)))
		}

		// Pass h() into the module so JSX works.
		hFn := vm.ToValue(hFunc)

		_, err = fn(goja.Undefined(), modExports, mod, vm.ToValue(modRequire), hFn)
		if err != nil {
			panic(vm.ToValue(fmt.Sprintf("require: execution error in %q: %v", modPath, err)))
		}

		// module.exports takes precedence (CommonJS convention).
		result := mod.Get("exports")
		cache[resolved] = result
		return result
	}
}

// resolveModulePath resolves a require path relative to baseDir.
func resolveModulePath(baseDir, modPath string) string {
	// If it starts with ./ or ../, it's a relative path.
	if modPath[0] == '.' {
		candidate := filepath.Join(baseDir, modPath)

		// Try exact path first.
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		// Try adding extensions.
		for _, ext := range []string{".tsx", ".ts", ".jsx", ".js", ".json"} {
			withExt := candidate + ext
			if _, err := os.Stat(withExt); err == nil {
				return withExt
			}
		}

		// Try index files in directory.
		for _, ext := range []string{".tsx", ".ts", ".jsx", ".js"} {
			idx := filepath.Join(candidate, "index"+ext)
			if _, err := os.Stat(idx); err == nil {
				return idx
			}
		}

		return candidate // will fail when reading
	}

	return modPath
}
