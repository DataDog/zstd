#!/usr/bin/env bash

VERSION="1.4.3"

# Remove old artefact before updating.
#rm -rf zstd-$VERSION.tar.gz
#rm -rf zstd-$VERSION.tar.gz.sha256

# Download $VERSION of the zstd lib.
#curl "https://github.com/facebook/zstd/releases/download/v$VERSION/zstd-$VERSION.tar.gz" -L -o zstd-$VERSION.tar.gz
#curl "https://github.com/facebook/zstd/releases/download/v$VERSION/zstd-$VERSION.tar.gz.sha256" -L -o zstd-$VERSION.tar.gz.sha256

cat zstd-$VERSION.tar.gz.sha256 | sha256sum --check --status

# get last command status
checksum_ok=$?
if [ $checksum_ok -eq 0 ]; then
    echo "Checksum OK"
else
    echo "Checksum FAIL"
    exit 1
fi

echo "Extracting tar.gz"
tar xf zstd-$VERSION.tar.gz
echo "Extraction done"

# Copy all the file listed in update.txt from the new $VERSION folder.
echo "Copy all the file from update.txt"
cp $(cat update.txt | sed "s#./lib#./zstd-$VERSION/lib#g") .

echo "Go build"
output=`go build 2>&1` # redirect stderr to stdout to be able to grep after.

# This loop is for copying every missing file from the compilation error messages.
# Since we do not need, nor want all the *.c and *.h file and I didn't find any better way of doing it (in cgo)
while echo $output | grep -q "file not found"; do
    filename=`echo $output | sed -n "s/.*'\(.*\)'.*/\1/p"`
    echo "$filename file not found"
    path_to_filename=`find ./zstd-$VERSION/lib -name $filename` # get the complete path of the file based on its name
    echo "Full path is : $path_to_filename"
    cp $path_to_filename .
    echo "Copied $path_to_filename to current dir"
    output=`go build 2>&1`
    echo "Output: $output"
done

echo "Updating README.md"
# update the README version. Match "v1.2.3"
sed -E "s#(v[0-9]+\.[0-9]+\.[0-9]+)#v$VERSION#g" README.md -i
