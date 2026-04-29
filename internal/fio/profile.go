package fio

import (
	"fmt"
	"os"
	"strings"
)

// BuildJobfile writes the jobfile content for a profile to a temporary
// file, substituting the <TARGET> placeholder with the given target path
// when present. Returns the path of the written file. The caller is
// responsible for removing it after the run.
//
// The seeded benchere fio profiles already carry a hard-coded filename=
// inside the [global] section, so by default we simply write configContent
// verbatim. Custom user profiles may use the <TARGET> placeholder to keep
// the jobfile portable; if any target overrides are passed, we replace
// the first occurrence of "<TARGET>" with the first target.
func BuildJobfile(profileName, configContent string, targets []string) (string, error) {
	out := configContent
	if strings.Contains(out, "<TARGET>") && len(targets) > 0 {
		out = strings.ReplaceAll(out, "<TARGET>", targets[0])
	}
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}

	tmp, err := os.CreateTemp("", "benchere-fio-"+sanitizeName(profileName)+"-*.fio")
	if err != nil {
		return "", fmt.Errorf("write jobfile: %w", err)
	}
	if _, err := tmp.WriteString(out); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("write jobfile content: %w", err)
	}
	tmp.Close()
	return tmp.Name(), nil
}

// sanitizeName keeps only [a-zA-Z0-9-_.] in a profile name so it is safe
// to splice into a temp filename.
func sanitizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
