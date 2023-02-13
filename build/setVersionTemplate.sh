#!/bin/sh
#Called by go generate
#Sets the version numbers in the template automatically
sed -i 's/{{define "version"}}.*{{end}}/{{define "version"}}'$1'{{end}}/g' ../../internal/webserver/web/templates/string_constants.tmpl
echo "Updated version in web template"
sed -i 's/{{define "js_admin_version"}}.*{{end}}/{{define "js_admin_version"}}'$2'{{end}}/g' ../../internal/webserver/web/templates/string_constants.tmpl
sed -i 's/{{define "js_dropzone_version"}}.*{{end}}/{{define "js_dropzone_version"}}'$3'{{end}}/g' ../../internal/webserver/web/templates/string_constants.tmpl
sed -i 's/{{define "js_e2eversion"}}.*{{end}}/{{define "js_e2eversion"}}'$4'{{end}}/g' ../../internal/webserver/web/templates/string_constants.tmpl
echo "Updated JS version numbers"
