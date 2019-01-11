#!/bin/bash

if [ "${GIMME_OS}" = "windows" ]; then
    zip remoteSwitch-v$TRAVIS_TAG-$GIMME_OS-$GIMME_ARCH.zip remoteSwitch.exe
else
    tar -cvzf remoteSwitch-v$TRAVIS_TAG-$GIMME_OS-$GIMME_ARCH.tar.gz remoteSwitch
fi
