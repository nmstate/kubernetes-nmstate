default: install

build_dir=build
dest_dir=$(build_dir)/$(shell sed -En "s/^baseurl: \"(.*)\"/\1/gp" _config.yaml)

.PHONY: help
help:
	@egrep '(^\S)|^$$' Makefile

.PHONY: install
install:
	bundle config set --local path vendor/bundle
	bundle install

.PHONY: upgrade
upgrade:
	bundle update

.PHONY: build
build:
	rm -rf $(dest_dir)
	bundle exec jekyll build --trace --source . --destination $(dest_dir)

.PHONY: check
check: build
	bundle exec htmlproofer --disable-external --empty-alt-ignore --only-4xx --log-level :debug $(build_dir)

.PHONY: serve
serve:
	rm -rf $(dest_dir)
	bundle exec jekyll serve --source . --destination $(dest_dir) --livereload --trace
