package scanx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/warbletech/shrikeops/scanner/pkg/scan"
)

const helmTimeout = 2 * time.Minute

var (
	errOutsideRoot    = errors.New("path is outside the allowed scan root")
	errInvalidChart   = errors.New("invalid chart path")
	errInvalidRelease = errors.New("invalid release name")
	errInvalidSetKey  = errors.New("invalid set key")
	errInvalidValues  = errors.New("invalid values file")
)

// Manifest runs ShrikeOps on a path or inline YAML.
func Manifest(path, manifest, source string) (string, error) {
	var content string
	src := source
	if path != "" {
		root, err := scanRoot()
		if err != nil {
			return "", err
		}
		rel, err := containedRel(root, path)
		if err != nil {
			return "", err
		}
		b, err := readLocal(root, rel)
		if err != nil {
			return "", err
		}
		content = string(b)
		if src == "" {
			src = filepath.Base(rel)
		}
	} else if manifest != "" {
		content = manifest
		if src == "" {
			src = "inline.yaml"
		}
	} else {
		return "", fmt.Errorf("provide path or manifest")
	}

	rep, err := scan.Run(scan.Request{Manifest: content, Source: src})
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		return "", err
	}
	out, err := Normalize(raw)
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// scanRoot is STARLING_SCAN_ROOT, or the working directory when that is unset.
// The result is absolute and symlink-free.
func scanRoot() (string, error) {
	root := os.Getenv("STARLING_SCAN_ROOT")
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve scan root: %w", err)
		}
		root = wd
	}
	return canonicalDir(root)
}

// chartRoot is STARLING_CHART_ROOT, otherwise the scan root.
func chartRoot() (string, error) {
	if root := os.Getenv("STARLING_CHART_ROOT"); root != "" {
		return canonicalDir(root)
	}
	return scanRoot()
}

func canonicalDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root is not a directory")
	}
	return real, nil
}

// containedRel returns a lexical path relative to root.
// Absolute paths are accepted only after both sides are canonicalized and the
// candidate is still inside root. Relative paths must already be local.
// filepath.IsLocal is the containment check; this does not use strings.HasPrefix.
func containedRel(root, p string) (string, error) {
	if p == "" || strings.ContainsRune(p, 0) {
		return "", errOutsideRoot
	}
	if filepath.IsAbs(p) {
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", errOutsideRoot
		}
		real, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return "", fmt.Errorf("resolve path: %w", err)
		}
		rel, err := filepath.Rel(root, real)
		if err != nil || !filepath.IsLocal(rel) {
			return "", errOutsideRoot
		}
		return rel, nil
	}
	rel := filepath.Clean(p)
	if !filepath.IsLocal(rel) {
		return "", errOutsideRoot
	}
	return rel, nil
}

// readLocal reads name inside root. name must be relative.
// filepath.IsLocal is applied in this function so the opened name is the
// sanitized value. os.Root rejects absolute names, "..", and symlinks that
// leave the root, so the user path is never passed to os.ReadFile.
func readLocal(root, name string) ([]byte, error) {
	if filepath.IsLocal(name) {
		r, err := os.OpenRoot(root)
		if err != nil {
			return nil, fmt.Errorf("open scan root: %w", err)
		}
		defer r.Close()
		f, err := r.Open(name)
		if err != nil {
			return nil, fmt.Errorf("read path: %w", err)
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("read path: %w", err)
		}
		return b, nil
	}
	return nil, errOutsideRoot
}

// Helm renders a chart with helm template then scans.
// chartPath and values files must resolve inside the chart root. The process
// is a fixed helm executable with separate arguments, never a shell string.
func Helm(chartPath string, valuesFiles []string, set map[string]string, release string) (string, error) {
	if release == "" {
		release = "scan"
	}
	if !validRelease(release) {
		return "", errInvalidRelease
	}
	root, err := chartRoot()
	if err != nil {
		return "", err
	}
	chartArg, err := chartArgument(root, chartPath)
	if err != nil {
		return "", err
	}
	valueArgs := make([]string, 0, len(valuesFiles))
	for _, vf := range valuesFiles {
		arg, err := valuesArgument(root, vf)
		if err != nil {
			return "", err
		}
		valueArgs = append(valueArgs, arg)
	}
	for k := range set {
		if !validSetKey(k) {
			return "", errInvalidSetKey
		}
	}
	helmBin, err := exec.LookPath("helm")
	if err != nil {
		return "", fmt.Errorf("helm not found on PATH (required for scan_helm)")
	}
	args := []string{"template", release, chartArg}
	for _, vf := range valueArgs {
		args = append(args, "-f", vf)
	}
	for k, v := range set {
		args = append(args, "--set", k+"="+v)
	}
	ctx, cancel := context.WithTimeout(context.Background(), helmTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, helmBin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("helm template: %w\n%s", err, string(out))
	}
	return Manifest("", string(out), filepath.Base(chartArg))
}

// chartArgument checks the chart directory inside root and returns the path
// helm should receive. That path is rebuilt from the canonical root and a
// local relative name, not the raw user string.
func chartArgument(root, chartPath string) (string, error) {
	if chartPath == "" || strings.ContainsRune(chartPath, 0) || rejectFlagPath(chartPath) {
		return "", errInvalidChart
	}
	rel, err := containedRel(root, chartPath)
	if err != nil {
		return "", errInvalidChart
	}
	if rejectFlagPath(rel) || !filepath.IsLocal(rel) {
		return "", errInvalidChart
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return "", fmt.Errorf("open chart root: %w", err)
	}
	defer r.Close()
	if filepath.IsLocal(rel) {
		info, err := r.Stat(rel)
		if err != nil || !info.IsDir() {
			return "", errInvalidChart
		}
		chartYAML := filepath.Join(rel, "Chart.yaml")
		if !filepath.IsLocal(chartYAML) {
			return "", errInvalidChart
		}
		fi, err := r.Stat(chartYAML)
		if err != nil || fi.IsDir() {
			return "", fmt.Errorf("chart is missing Chart.yaml")
		}
		return canonicalInside(root, rel)
	}
	return "", errInvalidChart
}

// canonicalInside resolves rel under root and rejects a symlink that leaves root.
func canonicalInside(root, rel string) (string, error) {
	if !filepath.IsLocal(rel) {
		return "", errOutsideRoot
	}
	real, err := filepath.EvalSymlinks(filepath.Join(root, rel))
	if err != nil {
		return "", err
	}
	back, err := filepath.Rel(root, real)
	if err != nil || !filepath.IsLocal(back) {
		return "", errOutsideRoot
	}
	return real, nil
}

func valuesArgument(root, valuesPath string) (string, error) {
	if valuesPath == "" || strings.ContainsRune(valuesPath, 0) || rejectFlagPath(valuesPath) {
		return "", errInvalidValues
	}
	rel, err := containedRel(root, valuesPath)
	if err != nil {
		return "", errInvalidValues
	}
	if rejectFlagPath(rel) || !filepath.IsLocal(rel) {
		return "", errInvalidValues
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return "", fmt.Errorf("open chart root: %w", err)
	}
	defer r.Close()
	if filepath.IsLocal(rel) {
		info, err := r.Stat(rel)
		if err != nil || info.IsDir() {
			return "", errInvalidValues
		}
		real, err := canonicalInside(root, rel)
		if err != nil {
			return "", errInvalidValues
		}
		return real, nil
	}
	return "", errInvalidValues
}

// rejectFlagPath reports paths helm would treat as flags.
func rejectFlagPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "-") {
		return true
	}
	for _, elem := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if strings.HasPrefix(elem, "-") {
			return true
		}
	}
	return false
}

func validRelease(name string) bool {
	if name == "" || strings.HasPrefix(name, "-") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func validSetKey(key string) bool {
	if key == "" || strings.HasPrefix(key, "-") || strings.ContainsRune(key, 0) {
		return false
	}
	return !strings.ContainsAny(key, "=\r\n")
}

// ClusterExport is a stub that tells the agent how to export (full export needs kubectl).
func ClusterExport(namespace string, kinds []string) (string, error) {
	if len(kinds) == 0 {
		kinds = []string{"deploy", "sts", "ds", "cronjob"}
	}
	nsFlag := ""
	if namespace != "" {
		nsFlag = "-n " + namespace
	} else {
		nsFlag = "-A"
	}
	kindList := strings.Join(kinds, ",")
	script := fmt.Sprintf(`# Export live workloads then scan:
kubectl get %s %s -o yaml > /tmp/cluster-export.yaml
# Then call scan_manifest with path=/tmp/cluster-export.yaml
`, kindList, nsFlag)
	// Best-effort: if kubectl exists, do it
	if _, err := exec.LookPath("kubectl"); err != nil {
		return script + "\nkubectl not found; run the commands above then scan_manifest.", nil
	}
	args := []string{"get", kindList, "-o", "yaml"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	} else {
		args = append(args, "-A")
	}
	out, err := exec.Command("kubectl", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl export: %w\n%s\n%s", err, string(out), script)
	}
	return Manifest("", string(out), "cluster-export")
}
