package playerid

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

type CharRef struct {
	Region string `json:"region"`
	Realm  string `json:"realm"`
	Name   string `json:"name"`
}

func (c CharRef) Canonical() CharRef {
	return CharRef{
		Region: strings.ToLower(strings.TrimSpace(c.Region)),
		Realm:  strings.ToLower(strings.TrimSpace(c.Realm)),
		Name:   strings.ToLower(strings.TrimSpace(c.Name)),
	}
}

func (c CharRef) String() string {
	return c.Region + "/" + c.Realm + "/" + c.Name
}

type ManualLink struct {
	A    CharRef `json:"a"`
	B    CharRef `json:"b"`
	Note string  `json:"note,omitempty"`
}

type manualLinksFile struct {
	Links []ManualLink `json:"links"`
}

// missing file is not an error - pipeline degrades gracefully
func LoadManualLinks(path string) ([]ManualLink, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var f manualLinksFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	out := make([]ManualLink, 0, len(f.Links))
	for _, l := range f.Links {
		ca, cb := l.A.Canonical(), l.B.Canonical()
		if ca.Region == "" || ca.Realm == "" || ca.Name == "" {
			continue
		}
		if cb.Region == "" || cb.Realm == "" || cb.Name == "" {
			continue
		}
		if ca == cb {
			continue
		}
		out = append(out, l)
	}
	return out, nil
}

func WriteManualLinks(path string, links []ManualLink) error {
	// normalise pair direction so (A, B) and (B, A) sort the same
	sorted := make([]ManualLink, len(links))
	copy(sorted, links)
	for i, l := range sorted {
		ca, cb := l.A.Canonical(), l.B.Canonical()
		if cb.String() < ca.String() {
			sorted[i].A, sorted[i].B = l.B, l.A
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		ai := sorted[i].A.Canonical().String() + "|" + sorted[i].B.Canonical().String()
		aj := sorted[j].A.Canonical().String() + "|" + sorted[j].B.Canonical().String()
		return ai < aj
	})

	body, err := json.MarshalIndent(manualLinksFile{Links: sorted}, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func SameLink(x, y ManualLink) bool {
	xa, xb := x.A.Canonical(), x.B.Canonical()
	ya, yb := y.A.Canonical(), y.B.Canonical()
	return (xa == ya && xb == yb) || (xa == yb && xb == ya)
}
