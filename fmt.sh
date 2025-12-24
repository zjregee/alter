#!/bin/bash

go mod tidy
goimports -w .
go vet ./...
