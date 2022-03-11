#!/bin/sh
#Called by go generate
#Creates the coverage for the README.md file if gopherbadger is installed

cd ../
which gopherbadger > /dev/null
if [ $? -eq 0 ]; then
	gopherbadger -png=false -md=README.md -tags "test,awsmock" > /dev/null
	rm coverage.out
	echo "Updated coverage in readme file"
else
	echo "Gopherbadger not installed, not updating coverage"
fi
