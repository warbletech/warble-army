package scanx

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/warbletech/shrikeops/scanner/pkg/scan"
)

// Manifest runs ShrikeOps on a path or inline YAML.
func Manifest(path, manifest, source string) (string, error) {
	var content string
	src := source
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read path: %w", err)
		}
		content = string(b)
		if src == "" {
			src = filepath.Base(path)
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
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Helm renders a chart with helm template then scans.
func Helm(chartPath string, valuesFiles []string, set map[string]string, release string) (string, error) {
	if release == "" {
		release = "scan"
	}
	if _, err := exec.LookPath("helm"); err != nil {
		return "", fmt.Errorf("helm not found on PATH (required for scan_helm)")
	}
	args := []string{"template", release, chartPath}
	for _, vf := range valuesFiles {
		args = append(args, "-f", vf)
	}
	for k, v := range set {
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, v))
	}
	out, err := exec.Command("helm", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("helm template: %w\n%s", err, string(out))
	}
	return Manifest("", string(out), filepath.Base(chartPath))
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
