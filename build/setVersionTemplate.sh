#!/bin/sh
#Called by go generate
#Sets the version number in the template automatically
sed -i 's/{{define "version"}}.*{{end}}/{{define "version"}}'$1'{{end}}/g' ../../internal/webserver/web/templates/string_constants.tmpl
echo "Set version in web template"
