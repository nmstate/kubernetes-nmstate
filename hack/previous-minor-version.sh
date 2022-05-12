#!/bin/bash

version_format="^v[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+$"

function git_tags_desc()
{
    git tag -l --sort=-v:refname | egrep $version_format
}

function latest_minor()
{
    git_tags_desc | head -n1
}

latest_minor
