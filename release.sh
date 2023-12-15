#!/bin/bash

# performs project build and creates release files for different architectures

if [ -z "$GITHUB_REPOSITORY" ]; then
    binname=$(basename $(pwd))
else
    binname=$(echo $GITHUB_REPOSITORY | awk -F'/' '{print $NF}')
fi

TAG=${GITHUB_REF#refs/*/}
if [ -z "$TAG" ]; then
    TAG="dev"
fi

BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$BRANCH" = "HEAD" ]; then
    BRANCH=""
fi

RELEASE_DIR=build
BUILDTIMESTAMP=$(date +"%Y-%m-%d_%H%M%S")
COMMIT=$(git rev-parse --short HEAD)
os="linux"
arch="amd64 arm64"
osarch="!darwin/arm !darwin/386"
output="$binname-{{.OS}}-{{.Arch}}-$TAG"
ldflags="-X main.gitTag=${TAG} -X main.gitCommit=${COMMIT} -X main.gitBranch=${BRANCH} -X main.buildTimestamp=${BUILDTIMESTAMP}"

go install github.com/mitchellh/gox@latest
mkdir -p $RELEASE_DIR
rm -f ./$RELEASE_DIR/*

pushd src || exit 1
CGO_ENABLED=0 gox -ldflags "$ldflags" -os="$os" -arch="$arch" -osarch="$osarch" -output="$output"
popd

mkdir -p $RELEASE_DIR/config

pushd test || exit 1
for file in *.yml; do
    if [ -f "$file" ]; then
        cp $file ../$RELEASE_DIR/config/example-$file
    fi
done
popd

cd src
for file in $binname-*; do
    if [ -f "$file" ]; then
        arch_suffix="${file#$binname-}"
        mv -f $file ../$RELEASE_DIR/$binname
        pushd ../$RELEASE_DIR
        zip $file.zip $binname config/*
        popd
    fi
done
cd ..
rm -fr $RELEASE_DIR/config $RELEASE_DIR/$binname

cd $RELEASE_DIR
sha256sum *.zip
