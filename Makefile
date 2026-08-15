.PHONY: setup test check validate scrape backfill charts dashboard dashboard-build site deploy-pages all

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

dashboard: dashboard-build

dashboard-build:
	go run ./cmd/dashboard $(ARGS)

site:
	go run ./cmd/charts --png $(ARGS)
	go run ./cmd/dashboard --site $(ARGS)

deploy-pages: site
	@tmp=$$(mktemp -d); \
	remote=$${DEPLOY_REMOTE:-$$(git remote get-url origin)}; \
	if git clone --quiet --branch gh-pages --single-branch $$remote $$tmp 2>/dev/null; then \
		git -C $$tmp rm -r --ignore-unmatch . >/dev/null; \
	else \
		git init --quiet $$tmp; \
		git -C $$tmp remote add origin $$remote; \
		git -C $$tmp checkout --orphan gh-pages 2>/dev/null || git -C $$tmp checkout -b gh-pages; \
	fi; \
	cp -R public/. $$tmp/; \
	git -C $$tmp add -A; \
	if git -C $$tmp diff --cached --quiet; then \
		echo "gh-pages ist bereits aktuell"; \
	else \
		git -C $$tmp commit -m "Deploy GitHub Pages site"; \
		git -C $$tmp push -u origin gh-pages; \
	fi; \
	rm -rf $$tmp

all:
	go run ./cmd/scrape --postcodes all
	go run ./cmd/charts --png
	go run ./cmd/dashboard
