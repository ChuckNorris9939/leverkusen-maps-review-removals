package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func normalizedSiteURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/"
	}
	return strings.TrimRight(value, "/") + "/"
}

func buildSite(a args) error {
	out := strings.TrimSpace(a.SiteOutput)
	if out == "" {
		out = "public"
	}
	chartsDir := filepath.Dir(a.Output)
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(out, "charts"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(out, "data"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, ".nojekyll"), nil, 0o644); err != nil {
		return err
	}
	if strings.TrimSpace(a.SiteDomain) != "" {
		if err := os.WriteFile(filepath.Join(out, "CNAME"), []byte(strings.TrimSpace(a.SiteDomain)+"\n"), 0o644); err != nil {
			return err
		}
	}
	if err := copyFile(a.Output, filepath.Join(out, "index.html")); err != nil {
		return err
	}
	for _, page := range legalPageFiles {
		src := filepath.Join(chartsDir, page)
		if _, err := os.Stat(src); err == nil {
			if err := copyFile(src, filepath.Join(out, page)); err != nil {
				return err
			}
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	entries, err := os.ReadDir(chartsDir)
	if err != nil {
		return err
	}
	legal := map[string]bool{}
	for _, page := range legalPageFiles {
		legal[page] = true
	}
	for _, entry := range entries {
		if entry.IsDir() || legal[entry.Name()] {
			continue
		}
		if err := copyFile(filepath.Join(chartsDir, entry.Name()), filepath.Join(out, "charts", entry.Name())); err != nil {
			return err
		}
	}
	for _, path := range []string{"output/metadata.json", "output/places.csv"} {
		if err := copyFile(path, filepath.Join(out, "data", filepath.Base(path))); err != nil {
			return err
		}
	}
	return writeSiteMeta(a, out)
}

func writeSiteMeta(a args, out string) error {
	siteURL := normalizedSiteURL(a.SiteURL)
	if siteURL == "/" {
		return nil
	}
	if err := os.WriteFile(filepath.Join(out, "robots.txt"), []byte(fmt.Sprintf("User-agent: *\nAllow: /\nSitemap: %ssitemap.xml\n", siteURL)), 0o644); err != nil {
		return err
	}
	lastmod := time.Now().UTC().Format("2006-01-02")
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	fmt.Fprintf(&b, "  <url><loc>%s</loc><lastmod>%s</lastmod><changefreq>weekly</changefreq><priority>1.0</priority></url>\n", siteURL, lastmod)
	for _, page := range legalPageFiles {
		if _, err := os.Stat(filepath.Join(out, page)); err == nil {
			fmt.Fprintf(&b, "  <url><loc>%s%s</loc><lastmod>%s</lastmod><changefreq>monthly</changefreq><priority>0.5</priority></url>\n", siteURL, page, lastmod)
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	fmt.Fprintf(&b, "  <url><loc>%scharts/nuernberg_most_removed.html</loc><lastmod>%s</lastmod><changefreq>weekly</changefreq><priority>0.6</priority></url>\n", siteURL, lastmod)
	b.WriteString("</urlset>\n")
	return os.WriteFile(filepath.Join(out, "sitemap.xml"), []byte(b.String()), 0o644)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
