#! /bin/sh

set -e

VERSION=$(cat VERSION)

# Generate the pkg/version.go
sed -i "s/const Version.*/const Version = \"$VERSION\"/g" pkg/version.go
git add pkg/version.go

# Generate the plugin.yaml
sed -i "s/version:.*$/version: \"$VERSION\"/g" plugin.yaml
git add plugin.yaml
