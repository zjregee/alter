#!/bin/bash

go mod tidy
goimports -w .
go vet ./...
golangci-lint run ./...

cd frontend
if [ ! -d "node_modules" ]; then
    npm install
fi
npm run format
npm run lint
cd ..
