default: serve

.PHONY: help
help:
	@egrep '(^\S)|^$$' Makefile

.PHONY: install
install:
	npm install

# Move legacy Ruby vendor/ aside if present, as it conflicts with Go modules.
.PHONY: serve
serve: install
	@if [ -d vendor ]; then mv vendor vendor.bak; fi
	hugo server --source .; ret=$$?; \
	if [ -d vendor.bak ]; then mv vendor.bak vendor; fi; \
	exit $$ret

.PHONY: build
build: install
	@if [ -d vendor ]; then mv vendor vendor.bak; fi
	hugo --source . --destination build/; ret=$$?; \
	if [ -d vendor.bak ]; then mv vendor.bak vendor; fi; \
	exit $$ret

.PHONY: check
check: build
	htmltest
