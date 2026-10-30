package scanx

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	sentinelOutside = "SENTINEL-OUTSIDE-FILE"
	validManifest   = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: demo\n"
)

func TestManifestReadsFileInsideScanRoot(t *testing.T) {
	root, _ := makeScanRoot(t)
	if err := os.WriteFile(filepath.Join(root, "deploy.yaml"), []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Manifest("deploy.yaml", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"score"`) {
		t.Fatalf("report missing score: %s", out)
	}
}

func TestManifestAbsoluteInsideRoot(t *testing.T) {
	root, _ := makeScanRoot(t)
	path := filepath.Join(root, "deploy.yaml")
	if err := os.WriteFile(path, []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Manifest(path, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"score"`) {
		t.Fatalf("report missing score: %s", out)
	}
}

func TestManifestRejectsTraversal(t *testing.T) {
	root, outside := makeScanRoot(t)
	secret := filepath.Join(outside, "secret.yaml")
	writeSecret(t, secret)
	got, err := Manifest(filepath.Join("..", "outside", "secret.yaml"), "", "")
	assertRejectedRead(t, root, got, err)
}

func TestManifestRejectsAbsoluteOutside(t *testing.T) {
	root, outside := makeScanRoot(t)
	secret := filepath.Join(outside, "secret.yaml")
	writeSecret(t, secret)
	got, err := Manifest(secret, "", "")
	assertRejectedRead(t, root, got, err)
}

func TestManifestRejectsSymlinkOutside(t *testing.T) {
	root, outside := makeScanRoot(t)
	secret := filepath.Join(outside, "secret.yaml")
	writeSecret(t, secret)
	link := filepath.Join(root, "escape")
	if err := linkOutside(outside, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	got, err := Manifest(filepath.Join("escape", "secret.yaml"), "", "")
	assertRejectedRead(t, root, got, err)
}

// linkOutside points link at target. A file symlink is used when the OS allows
// it. Windows falls back to a directory junction, which does not need the
// symlink privilege and still leaves the scan root.
func linkOutside(target, link string) error {
	if err := os.Symlink(target, link); err == nil {
		return nil
	}
	cmd := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(out)) + ": " + err.Error())
	}
	return nil
}

func TestHelmValidChart(t *testing.T) {
	root := makeChartRoot(t)
	argsFile := installFakeHelm(t)
	chart := filepath.Join(root, "demo")
	if err := os.Mkdir(chart, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chart, "Chart.yaml"), []byte("apiVersion: v2\nname: demo\nversion: 0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Helm("demo", nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"score"`) {
		t.Fatalf("report missing score: %s", out)
	}
	logged := readFile(t, argsFile)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(realRoot, "demo")
	if !strings.Contains(logged, want) {
		t.Fatalf("helm args %q do not contain %q", logged, want)
	}
	if !strings.Contains(logged, "template") || !strings.Contains(logged, "scan") {
		t.Fatalf("helm args %q", logged)
	}
}

func TestHelmRejectsTraversal(t *testing.T) {
	root := makeChartRoot(t)
	argsFile := installFakeHelm(t)
	outside := filepath.Join(filepath.Dir(root), "outside-chart")
	writeChart(t, outside)
	_, err := Helm(filepath.Join("..", "outside-chart"), nil, nil, "scan")
	assertHelmNotRun(t, argsFile, err)
}

func TestHelmRejectsAbsoluteOutside(t *testing.T) {
	root := makeChartRoot(t)
	argsFile := installFakeHelm(t)
	outside := filepath.Join(filepath.Dir(root), "outside-chart")
	writeChart(t, outside)
	_, err := Helm(outside, nil, nil, "scan")
	assertHelmNotRun(t, argsFile, err)
}

func TestHelmRejectsFlagPaths(t *testing.T) {
	root := makeChartRoot(t)
	writeChart(t, filepath.Join(root, "demo"))
	for _, chart := range []string{"-rf", "--post-renderer", filepath.Join("demo", "-hidden")} {
		t.Run(chart, func(t *testing.T) {
			argsFile := installFakeHelm(t)
			_, err := Helm(chart, nil, nil, "scan")
			assertHelmNotRun(t, argsFile, err)
		})
	}
}

func TestHelmPassesInsideValuesFile(t *testing.T) {
	root := makeChartRoot(t)
	argsFile := installFakeHelm(t)
	writeChart(t, filepath.Join(root, "demo"))
	if err := os.WriteFile(filepath.Join(root, "values.yaml"), []byte("replicaCount: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Helm("demo", []string{"values.yaml"}, nil, "scan"); err != nil {
		t.Fatal(err)
	}
	logged := readFile(t, argsFile)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(realRoot, "values.yaml")
	if !strings.Contains(logged, want) {
		t.Fatalf("helm args %q do not contain %q", logged, want)
	}
}

func TestHelmRejectsValuesTraversal(t *testing.T) {
	root := makeChartRoot(t)
	argsFile := installFakeHelm(t)
	writeChart(t, filepath.Join(root, "demo"))
	secret := filepath.Join(filepath.Dir(root), "values.yaml")
	writeSecret(t, secret)
	_, err := Helm("demo", []string{filepath.Join("..", "values.yaml")}, nil, "scan")
	assertHelmNotRun(t, argsFile, err)
}

func TestHelmRejectsSymlinkChart(t *testing.T) {
	root := makeChartRoot(t)
	argsFile := installFakeHelm(t)
	outside := filepath.Join(filepath.Dir(root), "outside-chart")
	writeChart(t, outside)
	link := filepath.Join(root, "escape")
	if err := linkOutside(outside, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	_, err := Helm("escape", nil, nil, "scan")
	assertHelmNotRun(t, argsFile, err)
	_, err = Helm(link, nil, nil, "scan")
	assertHelmNotRun(t, argsFile, err)
}

func TestHelmRejectsSymlinkValues(t *testing.T) {
	root := makeChartRoot(t)
	argsFile := installFakeHelm(t)
	writeChart(t, filepath.Join(root, "demo"))
	outside := filepath.Join(filepath.Dir(root), "outside-values")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "values.yaml")
	writeSecret(t, secret)
	link := filepath.Join(root, "escape")
	if err := linkOutside(outside, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	_, err := Helm("demo", []string{filepath.Join("escape", "values.yaml")}, nil, "scan")
	assertHelmNotRun(t, argsFile, err)
	_, err = Helm("demo", []string{filepath.Join(link, "values.yaml")}, nil, "scan")
	assertHelmNotRun(t, argsFile, err)
}

func TestInlineManifestStillScans(t *testing.T) {
	out, err := Manifest("", validManifest, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"score"`) {
		t.Fatalf("report missing score: %s", out)
	}
}

func makeScanRoot(t *testing.T) (root, outside string) {
	t.Helper()
	parent := t.TempDir()
	root = filepath.Join(parent, "root")
	outside = filepath.Join(parent, "outside")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STARLING_SCAN_ROOT", root)
	return root, outside
}

func makeChartRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("STARLING_CHART_ROOT", root)
	t.Setenv("STARLING_SCAN_ROOT", root)
	return root
}

func writeSecret(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(sentinelOutside), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeChart(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("apiVersion: v2\nname: demo\nversion: 0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertRejectedRead(t *testing.T, root, got string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got report %s", got)
	}
	if !errors.Is(err, errOutsideRoot) && !strings.Contains(err.Error(), "read path") {
		t.Fatalf("error %v", err)
	}
	if strings.Contains(got, sentinelOutside) || strings.Contains(err.Error(), sentinelOutside) {
		t.Fatalf("outside file was read: %v %s", err, got)
	}
	if _, statErr := os.Stat(filepath.Join(root, "not-created")); statErr == nil {
		t.Fatal("unexpected file")
	}
}

func assertHelmNotRun(t *testing.T, argsFile string, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(argsFile); !errors.Is(statErr, os.ErrNotExist) {
		body, _ := os.ReadFile(argsFile)
		t.Fatalf("helm was executed: %s (%v)", body, err)
	}
}

func installFakeHelm(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	if err := os.Mkdir(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module fakehelm\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const src = `package main
import (
  "fmt"
  "os"
  "strings"
)
func main() {
  path := os.Getenv("HELM_ARGS_FILE")
  if path != "" {
    _ = os.WriteFile(path, []byte(strings.Join(os.Args, "\n")), 0o644)
  }
  fmt.Print("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: scanned\n")
}
`
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "helm.exe")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = srcDir
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake helm: %v\n%s", err, out)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	argsFile := filepath.Join(dir, "args.txt")
	t.Setenv("HELM_ARGS_FILE", argsFile)
	return argsFile
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
