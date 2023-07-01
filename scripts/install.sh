#!/usr/bin/env bash


# package=$1
package="github.com/perun-cloud-inc/perun_hammer/cmd/hammer_events"

if [[ -z "$package" ]]; then
  echo "usage: $0 <package-name>"
  exit 1
fi


package_split=(${package//\// })

package_name=${package_split[-1]}


platforms=["linux/amd64"]

for platform in "${platforms[@]}"
do
	platform_split=(${platform//\// })
	GOOS=${platform_split[0]}
	GOARCH=${platform_split[1]}

    output_name=$package_name'-'$GOOS'-'$GOARCH


    env GOOS=$GOOS GOARCH=$GOARCH go build -o $output_name $package
    if [ $? -ne 0 ]; then
   		echo 'An error has occurred! Aborting the script execution...'
		exit 1
	fi

done

