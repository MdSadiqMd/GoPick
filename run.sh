#!/bin/bash

export PATH="$(go env GOPATH)/bin:$PATH"

if ! command -v gopick &> /dev/null; then
    echo "gopick not found, installing..."
    go install .
    if [ $? -ne 0 ]; then
        echo "Installation failed. Please run: go install ."
        exit 1
    fi
fi

gopick
