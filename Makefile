.PHONY: setup test check validate scrape backfill charts dashboard dashboard-build open-dashboard site deploy-pages all

ifneq (,$(wildcard .env))
include .env
export
endif

SITE_DOMAIN ?= nuernberg-maps-review-removals.patwoz.dev
SITE_URL ?= https://$(SITE_DOMAIN)

setup:
	go mod download

test:
	go test ./...

check:
	go test ./...
	go run ./cmd/validate

validate:
	go run ./cmd/validate $(ARGS)

scrape:
	go run ./cmd/scrape $(ARGS)

backfill:
	go run ./cmd/backfill $(ARGS)

charts:
	go run ./cmd/charts $(ARGS)

dashboard: dashboard-build open-dashboard

dashboard-build:
	go run ./cmd/dashboard $(ARGS)

open-dashboard:
	@file="$$(pwd)/output/charts/nuernberg_dashboard.html"; \
	if command -v open >/dev/null 2>&1; then \
		open "$$file"; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		xdg-open "$$file" >/dev/null 2>&1 & \
	else \
		echo "Dashboard geschrieben: $$file"; \
	fi

site:
	go run ./cmd/charts --png $(ARGS)
	go run ./cmd/dashboard
	rm -rf public
	mkdir -p public/charts public/data
	touch public/.nojekyll
	echo "$(SITE_DOMAIN)" > public/CNAME
	printf "User-agent: *\nAllow: /\nSitemap: $(SITE_URL)/sitemap.xml\n" > public/robots.txt
	@lastmod="$$(date -u +%Y-%m-%d)"; \
	printf '%s\n' '<?xml version="1.0" encoding="UTF-8"?>' '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">' "  <url><loc>$(SITE_URL)/</loc><lastmod>$$lastmod</lastmod><changefreq>weekly</changefreq><priority>1.0</priority></url>" "  <url><loc>$(SITE_URL)/charts/nuernberg_most_removed.html</loc><lastmod>$$lastmod</lastmod><changefreq>weekly</changefreq><priority>0.6</priority></url>" '</urlset>' > public/sitemap.xml
	cp output/charts/nuernberg_dashboard.html public/index.html
	cp output/charts/* public/charts/
	cp output/metadata.json output/places.csv public/data/

deploy-pages: site
	@tmp=$$(mktemp -d); \
	remote=$${DEPLOY_REMOTE:-$$(git remote get-url origin)}; \
	git clone --quiet --branch gh-pages --single-branch $$remote $$tmp; \
	git -C $$tmp rm -r --ignore-unmatch . >/dev/null; \
	cp -R public/. $$tmp/; \
	git -C $$tmp add -A; \
	if git -C $$tmp diff --cached --quiet; then \
		echo "gh-pages ist bereits aktuell"; \
	else \
		git -C $$tmp commit -m "Deploy GitHub Pages site"; \
		git -C $$tmp push origin gh-pages; \
	fi; \
	rm -rf $$tmp

all:
	go run ./cmd/scrape --postcodes all
	go run ./cmd/charts --png
	go run ./cmd/dashboard
