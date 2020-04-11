#!/bin/bash

if [ "${GOOS}" = "windows" ]; then
    zip remoteSwitch-$TRAVIS_TAG-$GOOS-$GOARCH.zip remoteSwitch.exe
else
    tar -cvzf remoteSwitch-$TRAVIS_TAG-$GOOS-$GOARCH.tar.gz remoteSwitch
fi
