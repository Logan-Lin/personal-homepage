package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/flosch/pongo2/v6"
	"github.com/tdewolff/minify/v2"
	mincss "github.com/tdewolff/minify/v2/css"
	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

const (
	srcAssets   = "asset"
	srcContent  = "content"
	srcTemplate = "templates"
	srcData     = "data.yaml"
	outDir      = "dist"
)

func build() error {
	t0 := time.Now()

	data, err := loadYAML(srcData)
	if err != nil {
		return fmt.Errorf("load %s: %w", srcData, err)
	}

	lookup := buildItemLookup(data)
	if home, ok := data["home"].(map[string]interface{}); ok {
		resolveSubsections(home["research"], lookup)
		resolveSubsections(home["teaching"], lookup)
	}

	content, err := renderContent(srcContent)
	if err != nil {
		return fmt.Errorf("render content: %w", err)
	}

	for _, d := range []string{outDir, filepath.Join(outDir, "publications"), filepath.Join(outDir, "projects"), filepath.Join(outDir, "activities")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	assets, err := copyAssets(srcAssets, outDir)
	if err != nil {
		return fmt.Errorf("copy assets: %w", err)
	}

	loader, err := pongo2.NewLocalFileSystemLoader(srcTemplate)
	if err != nil {
		return err
	}
	ts := pongo2.NewSet("homepage", loader)

	year := time.Now().Year()
	render := func(tmplName, outPath string, ctx pongo2.Context) error {
		tmpl, err := ts.FromFile(tmplName)
		if err != nil {
			return fmt.Errorf("template %s: %w", tmplName, err)
		}
		ctx["year"] = year
		ctx["assets"] = assets
		out, err := tmpl.ExecuteBytes(ctx)
		if err != nil {
			return fmt.Errorf("render %s: %w", tmplName, err)
		}
		if err := os.WriteFile(outPath, out, 0o644); err != nil {
			return err
		}
		fmt.Printf("Generated %s\n", outPath)
		return nil
	}

	pages := []struct {
		tmpl string
		out  string
		home bool
	}{
		{"index.html", filepath.Join(outDir, "index.html"), true},
		{"publications.html", filepath.Join(outDir, "publications", "index.html"), false},
		{"projects.html", filepath.Join(outDir, "projects", "index.html"), false},
		{"activities.html", filepath.Join(outDir, "activities", "index.html"), false},
	}
	for _, p := range pages {
		ctx := pongo2.Context{"data": data, "is_home_page": p.home}
		if p.home {
			ctx["content"] = content
		}
		if err := render(p.tmpl, p.out, ctx); err != nil {
			return err
		}
	}

	if err := renderSiteIndex(ts, data); err != nil {
		return err
	}

	fmt.Printf("Static site generation complete in %s\n", time.Since(t0).Round(time.Millisecond))
	return nil
}

func loadYAML(path string) (map[string]interface{}, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func buildItemLookup(data map[string]interface{}) map[string]map[string]interface{} {
	lookup := map[string]map[string]interface{}{}
	sources := []struct {
		path    []string
		kind    string
		subtype string
	}{
		{[]string{"publications", "primary", "entries"}, "publication", "primary"},
		{[]string{"publications", "secondary", "entries"}, "publication", "secondary"},
		{[]string{"projects", "primary", "entries"}, "project", "primary"},
		{[]string{"projects", "secondary", "entries"}, "project", "secondary"},
		{[]string{"activities", "courses", "entries"}, "teaching", "primary"},
		{[]string{"activities", "supervision", "entries"}, "teaching", "primary"},
		{[]string{"activities", "presentations", "entries"}, "presentation", "secondary"},
	}
	for _, src := range sources {
		var node interface{} = data
		for _, k := range src.path {
			m, ok := node.(map[string]interface{})
			if !ok {
				node = nil
				break
			}
			node = m[k]
		}
		items, ok := node.([]interface{})
		if !ok {
			continue
		}
		for _, it := range items {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			id, ok := m["id"].(string)
			if !ok {
				continue
			}
			entry := make(map[string]interface{}, len(m)+2)
			for k, v := range m {
				entry[k] = v
			}
			entry["_type"] = src.kind
			entry["_subtype"] = src.subtype
			lookup[id] = entry
		}
	}
	return lookup
}

func resolveSubsections(node interface{}, lookup map[string]map[string]interface{}) {
	sections, ok := node.([]interface{})
	if !ok {
		return
	}
	for _, s := range sections {
		sub, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		ids, _ := sub["entries"].([]interface{})
		resolved := make([]map[string]interface{}, 0, len(ids))
		for _, id := range ids {
			sid, ok := id.(string)
			if !ok {
				continue
			}
			if item, ok := lookup[sid]; ok {
				resolved = append(resolved, item)
			} else {
				fmt.Printf("Warning: item '%s' not found in lookup\n", sid)
			}
		}
		sub["resolved_entries"] = resolved
	}
}

func renderContent(dir string) (map[string]string, error) {
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	md := goldmark.New()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := md.Convert(raw, &buf); err != nil {
			return nil, err
		}
		out[strings.TrimSuffix(name, ".md")] = buf.String()
	}
	return out, nil
}

func fingerprint(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:8]
}

func hashedName(filename string, content []byte) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	return fmt.Sprintf("%s.%s%s", base, fingerprint(content), ext)
}

type deferredAsset struct {
	srcPath, targetDir, relPath, filename string
}

func copyAssets(src, dst string) (map[string]string, error) {
	assets := map[string]string{}
	var deferred []deferredAsset

	cssMin := minify.New()
	cssMin.AddFunc("text/css", mincss.Minify)

	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relDir, _ := filepath.Rel(src, filepath.Dir(p))
		relDirSlash := filepath.ToSlash(relDir)
		filename := info.Name()
		var targetDir, relPath string
		if relDirSlash == "." {
			targetDir = dst
			relPath = filename
		} else {
			targetDir = filepath.Join(dst, relDir)
			relPath = path.Join(relDirSlash, filename)
		}
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}

		if strings.HasSuffix(filename, ".webmanifest") {
			deferred = append(deferred, deferredAsset{p, targetDir, relPath, filename})
			return nil
		}

		var content []byte
		if strings.HasSuffix(filename, ".css") {
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			minified, err := cssMin.Bytes("text/css", raw)
			if err != nil {
				return err
			}
			content = minified
		} else {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			content = b
		}

		outName := hashedName(filename, content)
		dstPath := filepath.Join(targetDir, outName)
		if err := os.WriteFile(dstPath, content, 0o644); err != nil {
			return err
		}
		if relDirSlash == "." {
			assets[relPath] = outName
		} else {
			assets[relPath] = path.Join(relDirSlash, outName)
		}
		fmt.Printf("Hashed %s -> %s\n", p, dstPath)
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, d := range deferred {
		raw, err := os.ReadFile(d.srcPath)
		if err != nil {
			return nil, err
		}
		text := string(raw)
		for orig, hashed := range assets {
			text = strings.ReplaceAll(text, "/"+orig, "/"+hashed)
		}
		content := []byte(text)
		outName := hashedName(d.filename, content)
		dstPath := filepath.Join(d.targetDir, outName)
		if err := os.WriteFile(dstPath, content, 0o644); err != nil {
			return nil, err
		}
		assets[d.relPath] = outName
		fmt.Printf("Hashed %s -> %s\n", d.srcPath, dstPath)
	}
	return assets, nil
}

func renderSiteIndex(ts *pongo2.TemplateSet, data map[string]interface{}) error {
	site, _ := data["site"].(map[string]interface{})
	baseURL, _ := site["baseUrl"].(string)
	today := time.Now().Format("2006-01-02")
	pages := []map[string]string{
		{"path": "/", "lastmod": today},
		{"path": "/publications/", "lastmod": today},
		{"path": "/projects/", "lastmod": today},
		{"path": "/activities/", "lastmod": today},
	}
	files := []struct {
		tmpl, out string
	}{
		{"sitemap.xml", filepath.Join(outDir, "sitemap.xml")},
		{"robots.txt", filepath.Join(outDir, "robots.txt")},
	}
	for _, f := range files {
		tmpl, err := ts.FromFile(f.tmpl)
		if err != nil {
			return err
		}
		out, err := tmpl.ExecuteBytes(pongo2.Context{"base_url": baseURL, "pages": pages})
		if err != nil {
			return err
		}
		if err := os.WriteFile(f.out, out, 0o644); err != nil {
			return err
		}
		fmt.Printf("Generated %s\n", f.out)
	}
	return nil
}

