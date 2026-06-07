package config

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// parseDotenv reads KEY=VALUE lines. Blank lines and # comments are ignored, an
// optional leading "export " is stripped, and surrounding single/double quotes
// are removed. Lines without '=' are skipped.
func parseDotenv(r io.Reader) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 {
			if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			}
		}
		if k != "" {
			out[k] = v
		}
	}
	return out
}

// dotenvFiles returns candidate .env paths. NYBBLE_ENV_FILE overrides discovery;
// otherwise ./.env (dev convenience) then ~/.config/nybble/.env (installed use).
func dotenvFiles() []string {
	if p := os.Getenv("NYBBLE_ENV_FILE"); p != "" {
		return []string{p}
	}
	var files []string
	if wd, err := os.Getwd(); err == nil {
		files = append(files, filepath.Join(wd, ".env"))
	}
	if dir, err := ConfigDir(); err == nil {
		files = append(files, filepath.Join(dir, ".env"))
	}
	return files
}

// LoadDotenv loads .env files into the process environment WITHOUT overriding
// variables that are already set. Precedence: real env > ./.env >
// ~/.config/nybble/.env. Missing files are not an error. Call once at startup.
func LoadDotenv() {
	for _, path := range dotenvFiles() {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		kv := parseDotenv(f)
		f.Close()
		for k, v := range kv {
			if _, ok := os.LookupEnv(k); !ok {
				os.Setenv(k, v)
			}
		}
	}
}
