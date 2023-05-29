#!/usr/bin/env bash
release_id=$1
platform=$2

platform_split=(${platform//\// })
GOOS=${platform_split[0]}
GOARCH=${platform_split[1]}
output_name='perunctl-'$GOOS'-'$GOARCH
if [ $GOOS = "windows" ]; then
    output_name+='.exe'
fi    
env GOOS=$GOOS GOARCH=$GOARCH go build -o perunctl
env GOOS=$GOOS GOARCH=$GOARCH go build -C cmd/hammer_events -o events
mv  cmd/hammer_events/events ./
if [ $? -ne 0 ]; then
        echo 'An error has occurred! Aborting the script execution...'
    exit 1
fi
zip $output_name.zip perunctl events
curl -L -X POST -H "Accept: application/vnd.github+json" -H "Authorization: Bearer $GHT" -H "X-GitHub-Api-Version: 2022-11-28" -H "Content-Type: application/octet-stream" https://uploads.github.com/repos/perun-cloud-inc/perunctl/releases/$release_id/assets?name=$output_name.zip --data-binary "@$output_name.zip"
echo "finish to upload $output_name"