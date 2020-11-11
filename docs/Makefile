default: install

help:
	@egrep '(^\S)|^$$' Makefile

install:
	bundle config set --local path vendor/bundle
	bundle install

upgrade:
	bundle update

s serve:
	bundle exec jekyll serve --source sample_site --destination build/ --livereload --trace
