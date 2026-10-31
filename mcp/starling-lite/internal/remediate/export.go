package remediate

// Shape is the exported view of a workload kind: canonical kind, apiVersion
// and the path to its pod spec (e.g. spec.template.spec).
type Shape struct {
	Kind        string
	APIVersion  string
	PodSpecPath []string
}

// ShapeFor returns the Shape for a supported workload kind.
func ShapeFor(kind string) (Shape, bool) {
	s, ok := shapeFor(kind)
	if !ok {
		return Shape{}, false
	}
	return Shape{Kind: s.kind, APIVersion: s.apiVersion, PodSpecPath: append([]string(nil), s.podSpecPath...)}, true
}

// HasBuiltInFix reports whether the deterministic rules (fixFor) handle issue.
func HasBuiltInFix(issue string) bool {
	_, ok := fixFor(normalizeIssue(issue))
	return ok
}

// KubectlPatchCmd is the suggested server-side dry-run command for a patch.
func KubectlPatchCmd(s Shape, name, namespace string) string {
	return kubectlCmd(kindShape{kind: s.Kind, apiVersion: s.APIVersion, podSpecPath: s.PodSpecPath},
		Request{Name: name, Namespace: namespace}, "strategic")
}
