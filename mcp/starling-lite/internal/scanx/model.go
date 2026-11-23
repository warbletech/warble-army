package scanx

// scan_model: picklescan-style static opcode check of model weight files.
// Never unpickles or executes anything; local file (inside the scan root) or
// base64 bytes only. Remote (Hugging Face) fetch is intentionally not
// implemented here.

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	RuleModelPickleRCE     = "AI-MODEL-PICKLE-RCE"
	RuleModelPickleUnknown = "AI-MODEL-PICKLE-UNKNOWN-GLOBAL"
	defaultModelMaxBytes   = 256 << 20
)

func init() {
	AIRules[RuleModelPickleRCE] = AIRule{RuleModelPickleRCE, "critical", "Pickle imports a dangerous callable (code execution on load)",
		"Do not load this file. Obtain .safetensors weights from a trusted, pinned source."}
	AIRules[RuleModelPickleUnknown] = AIRule{RuleModelPickleUnknown, "medium", "Pickle imports a callable outside the known-safe allowlist",
		"Review the listed globals; prefer .safetensors or torch.load(weights_only=True)."}
}

// dangerous module prefixes / exact globals (module.name).
var dangerousModules = []string{"os", "posix", "nt", "subprocess", "sys", "socket", "shutil", "runpy", "pty",
	"webbrowser", "importlib", "ctypes", "code", "commands", "pickle", "_pickle", "marshal", "multiprocessing",
	"asyncio", "requests", "urllib", "http", "ftplib", "telnetlib", "smtplib", "platform", "signal", "pdb", "bdb", "timeit"}
var dangerousBuiltins = map[string]bool{"eval": true, "exec": true, "compile": true, "open": true, "__import__": true,
	"getattr": true, "setattr": true, "delattr": true, "apply": true, "execfile": true, "input": true, "breakpoint": true, "globals": true, "locals": true, "vars": true}

var safeGlobals = map[string]bool{
	"collections.OrderedDict": true, "torch._utils._rebuild_tensor_v2": true, "torch._utils._rebuild_parameter": true,
	"torch._utils._rebuild_tensor": true, "torch._tensor._rebuild_from_type_v2": true, "torch.Size": true,
	"torch.device": true, "torch.float32": true, "torch.float16": true, "torch.bfloat16": true,
	"numpy.core.multiarray._reconstruct": true, "numpy._core.multiarray._reconstruct": true, "numpy.ndarray": true,
	"numpy.dtype": true, "numpy.core.multiarray.scalar": true, "_codecs.encode": true,
	"builtins.set": true, "builtins.frozenset": true, "builtins.slice": true, "builtins.complex": true, "builtins.bytearray": true,
	"__builtin__.set": true,
}

// ModelReport is the scan_model result.
type ModelReport struct {
	Source    string           `json:"source"`
	Format    string           `json:"format"`  // safetensors | pickle | torch-zip | unknown
	Verdict   string           `json:"verdict"` // safe | pickle-allowlisted | suspicious | unsafe | unparseable
	Globals   []string         `json:"globals,omitempty"`
	Dangerous []string         `json:"dangerous,omitempty"`
	Unknown   []string         `json:"unknown,omitempty"`
	Findings  []map[string]any `json:"findings"`
	Notes     []string         `json:"notes,omitempty"`
}

func modelMaxBytes() int64 {
	if v := os.Getenv("STARLING_MODEL_MAX_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return defaultModelMaxBytes
}

// ScanModel checks a local model file (path inside the scan root) or base64 bytes.
func ScanModel(path, b64, source string) (string, error) {
	var data []byte
	max := modelMaxBytes()
	switch {
	case path != "":
		root, err := scanRoot()
		if err != nil {
			return "", err
		}
		rel, err := containedRel(root, path)
		if err != nil {
			return "", err
		}
		if data, err = readLocalCapped(root, rel, max); err != nil {
			return "", err
		}
		if source == "" {
			source = filepath.Base(rel)
		}
	case b64 != "":
		if int64(len(b64)) > max*4/3+4 {
			return "", fmt.Errorf("bytes_base64 larger than STARLING_MODEL_MAX_BYTES")
		}
		var err error
		if data, err = base64.StdEncoding.DecodeString(b64); err != nil {
			return "", fmt.Errorf("bytes_base64: %w", err)
		}
		if source == "" {
			source = "inline"
		}
	default:
		return "", fmt.Errorf("provide path or bytes_base64")
	}
	rep := ScanModelBytes(data, source)
	b, err := json.MarshalIndent(rep, "", "  ")
	return string(b), err
}

// ScanModelBytes classifies model bytes without executing them.
func ScanModelBytes(data []byte, source string) ModelReport {
	r := ModelReport{Source: source, Findings: []map[string]any{}}
	switch {
	case isSafetensors(data):
		r.Format, r.Verdict = "safetensors", "safe"
		return r
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		r.Format = "torch-zip"
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			r.Verdict = "unparseable"
			r.Notes = append(r.Notes, "zip: "+err.Error())
			break
		}
		found := false
		for _, f := range zr.File {
			if !strings.HasSuffix(f.Name, ".pkl") {
				continue
			}
			found = true
			if f.UncompressedSize64 > uint64(len(data))*20+1<<20 {
				r.Notes = append(r.Notes, f.Name+": skipped (compression ratio too high)")
				r.Verdict = "unparseable"
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			b, _ := io.ReadAll(io.LimitReader(rc, int64(len(data))*20+1<<20))
			rc.Close()
			globals, err := pickleGlobals(b)
			if err != nil {
				r.Notes = append(r.Notes, f.Name+": "+err.Error())
				r.Verdict = "unparseable"
			}
			r.Globals = append(r.Globals, globals...)
		}
		if !found {
			r.Notes = append(r.Notes, "zip has no .pkl entries")
		}
	default:
		globals, err := pickleGlobals(data)
		r.Format = "pickle"
		if len(data) == 0 || bytes.IndexByte([]byte("\x80(c}]).NIK"), data[0]) < 0 {
			r.Format = "unknown"
		}
		if err != nil {
			r.Verdict = "unparseable"
			r.Notes = append(r.Notes, err.Error())
		}
		r.Globals = globals
	}
	r.Globals = uniqSorted(r.Globals)
	for _, g := range r.Globals {
		switch {
		case isDangerousGlobal(g):
			r.Dangerous = append(r.Dangerous, g)
		case !safeGlobals[g] && !isTorchStorage(g):
			r.Unknown = append(r.Unknown, g)
		}
	}
	o := obj{kind: "ModelFile", ns: "-", name: source}
	switch {
	case len(r.Dangerous) > 0:
		r.Verdict = "unsafe"
		r.Findings = append(r.Findings, finding(RuleModelPickleRCE, o, "", strings.Join(r.Dangerous, ", ")))
	case len(r.Unknown) > 0:
		if r.Verdict == "" {
			r.Verdict = "suspicious"
		}
		r.Findings = append(r.Findings, finding(RuleModelPickleUnknown, o, "", strings.Join(r.Unknown, ", ")))
	case r.Verdict == "":
		r.Verdict = "pickle-allowlisted"
	}
	if r.Verdict == "unparseable" && len(r.Dangerous) == 0 {
		r.Findings = append(r.Findings, finding(RuleModelPickleUnknown, o, "", "pickle stream could not be fully parsed; treat as untrusted"))
	}
	if r.Format != "safetensors" {
		r.Findings = append(r.Findings, finding(RuleModelPickle, o, "", r.Format))
	}
	return r
}

func isSafetensors(b []byte) bool {
	if len(b) < 10 {
		return false
	}
	n := binary.LittleEndian.Uint64(b[:8])
	if n == 0 || n > uint64(len(b)-8) || n > 100<<20 {
		return false
	}
	var hdr map[string]any
	return b[8] == '{' && json.Unmarshal(b[8:8+n], &hdr) == nil
}

func isTorchStorage(g string) bool {
	return strings.HasPrefix(g, "torch.") && strings.HasSuffix(g, "Storage")
}

func isDangerousGlobal(g string) bool {
	mod, name, _ := strings.Cut(g, ".")
	if i := strings.LastIndex(g, "."); i > 0 {
		mod, name = g[:i], g[i+1:]
	}
	if mod == "builtins" || mod == "__builtin__" || mod == "__builtins__" {
		return dangerousBuiltins[name]
	}
	root, _, _ := strings.Cut(mod, ".")
	for _, d := range dangerousModules {
		if root == d {
			return true
		}
	}
	return false
}

func uniqSorted(in []string) []string {
	m := map[string]bool{}
	for _, s := range in {
		m[s] = true
	}
	out := make([]string, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// pickleGlobals walks pickle opcodes (protocols 0-5) and returns the
// module.name of every GLOBAL / INST / STACK_GLOBAL. It never executes.
func pickleGlobals(b []byte) ([]string, error) {
	var globals []string
	var strs []string // recent pushed values: string or "\x00" for non-string
	memo := map[uint64]string{}
	push := func(s string) { strs = append(strs, s) }
	last := func() string {
		if len(strs) == 0 {
			return "\x00"
		}
		return strs[len(strs)-1]
	}
	i := 0
	need := func(n int) error {
		if n < 0 || i+n > len(b) {
			return fmt.Errorf("truncated pickle at offset %d", i)
		}
		return nil
	}
	readLine := func() (string, error) {
		j := bytes.IndexByte(b[i:], '\n')
		if j < 0 {
			return "", fmt.Errorf("unterminated line at offset %d", i)
		}
		s := string(b[i : i+j])
		i += j + 1
		return s, nil
	}
	fixed := func(n int) ([]byte, error) {
		if err := need(n); err != nil {
			return nil, err
		}
		v := b[i : i+n]
		i += n
		return v, nil
	}
	lenPrefixed := func(width int) ([]byte, error) {
		lb, err := fixed(width)
		if err != nil {
			return nil, err
		}
		var n uint64
		switch width {
		case 1:
			n = uint64(lb[0])
		case 4:
			n = uint64(binary.LittleEndian.Uint32(lb))
		case 8:
			n = binary.LittleEndian.Uint64(lb)
		}
		if n > uint64(len(b)) {
			return nil, fmt.Errorf("length %d exceeds data at offset %d", n, i)
		}
		return fixed(int(n))
	}
	stopped := false
	for i < len(b) {
		op := b[i]
		i++
		var err error
		switch op {
		case '(', '0', '1', '2', 'N', 'Q', 'R', 'a', 'b', 'd', '}', 'e', 'l', ']', 'o', 's', 't', ')', 'u',
			0x81, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8f, 0x90, 0x91, 0x92, 0x97, 0x98:
			if op != '0' && op != '2' {
				push("\x00")
			}
		case '.': // STOP
			stopped = true
			strs = strs[:0]
		case 'F', 'I', 'L', 'P':
			_, err = readLine()
			push("\x00")
		case 'S', 'V':
			var s string
			s, err = readLine()
			push(strings.Trim(s, "'\""))
		case 'g':
			var s string
			if s, err = readLine(); err == nil {
				n, _ := strconv.ParseUint(s, 10, 64)
				push(memoGet(memo, n))
			}
		case 'p':
			var s string
			if s, err = readLine(); err == nil {
				n, _ := strconv.ParseUint(s, 10, 64)
				memo[n] = last()
			}
		case 'c', 'i': // GLOBAL / INST: two lines
			var mod, name string
			if mod, err = readLine(); err == nil {
				if name, err = readLine(); err == nil {
					globals = append(globals, mod+"."+name)
					push("\x00")
				}
			}
		case 0x93: // STACK_GLOBAL
			if len(strs) >= 2 && strs[len(strs)-1] != "\x00" && strs[len(strs)-2] != "\x00" {
				globals = append(globals, strs[len(strs)-2]+"."+strs[len(strs)-1])
			} else {
				globals = append(globals, "?.?") // unresolved -> unknown
			}
			push("\x00")
		case 'J':
			_, err = fixed(4)
			push("\x00")
		case 'K':
			_, err = fixed(1)
			push("\x00")
		case 'M':
			_, err = fixed(2)
			push("\x00")
		case 'G':
			_, err = fixed(8)
			push("\x00")
		case 'h':
			var v []byte
			if v, err = fixed(1); err == nil {
				push(memoGet(memo, uint64(v[0])))
			}
		case 'j':
			var v []byte
			if v, err = fixed(4); err == nil {
				push(memoGet(memo, uint64(binary.LittleEndian.Uint32(v))))
			}
		case 'q':
			var v []byte
			if v, err = fixed(1); err == nil {
				memo[uint64(v[0])] = last()
			}
		case 'r':
			var v []byte
			if v, err = fixed(4); err == nil {
				memo[uint64(binary.LittleEndian.Uint32(v))] = last()
			}
		case 0x94: // MEMOIZE
			memo[uint64(len(memo))] = last()
		case 0x80: // PROTO
			_, err = fixed(1)
		case 0x95: // FRAME
			_, err = fixed(8)
		case 0x82:
			_, err = fixed(1)
			push("\x00")
		case 0x83:
			_, err = fixed(2)
			push("\x00")
		case 0x84:
			_, err = fixed(4)
			push("\x00")
		case 0x8a:
			_, err = lenPrefixed(1)
			push("\x00")
		case 0x8b:
			_, err = lenPrefixed(4)
			push("\x00")
		case 'T', 'B':
			_, err = lenPrefixed(4)
			push("\x00")
		case 'U', 'C':
			_, err = lenPrefixed(1)
			push("\x00")
		case 'X':
			var v []byte
			if v, err = lenPrefixed(4); err == nil {
				push(string(v))
			}
		case 0x8c:
			var v []byte
			if v, err = lenPrefixed(1); err == nil {
				push(string(v))
			}
		case 0x8d:
			var v []byte
			if v, err = lenPrefixed(8); err == nil {
				push(string(v))
			}
		case 0x8e, 0x96:
			_, err = lenPrefixed(8)
			push("\x00")
		default:
			if stopped {
				// trailing non-pickle data after a complete pickle (e.g. legacy torch tensor blobs)
				return globals, nil
			}
			return globals, fmt.Errorf("unknown opcode 0x%02x at offset %d", op, i-1)
		}
		if err != nil {
			return globals, err
		}
	}
	if !stopped {
		return globals, fmt.Errorf("pickle has no STOP opcode")
	}
	return globals, nil
}

func memoGet(m map[uint64]string, k uint64) string {
	if v, ok := m[k]; ok {
		return v
	}
	return "\x00"
}

// readLocalCapped reads a file inside root (os.Root confinement) up to max bytes.
func readLocalCapped(root, name string, max int64) ([]byte, error) {
	if !filepath.IsLocal(name) {
		return nil, errOutsideRoot
	}
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
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, fmt.Errorf("read path: %w", err)
	}
	if int64(len(b)) > max {
		return nil, fmt.Errorf("model file larger than STARLING_MODEL_MAX_BYTES=%d", max)
	}
	return b, nil
}
