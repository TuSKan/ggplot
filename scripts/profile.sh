#!/usr/bin/env bash
set -e

# Evaluates explicit native Go performance metrics wrapping heavy benchmark test loops reliably configuring bounds efficiently.
echo "Gathering explicit profile constraints targeting native metrics cleanly..."
mkdir -p profiles

go test -bench=. -benchmem -cpuprofile=profiles/cpu.prof -memprofile=profiles/mem.prof ./...

echo "Profiles captured reliably!"
echo "Evaluate explicitly executing smoothly:"
echo "  go tool pprof -http=:8080 profiles/cpu.prof"
echo "  go tool pprof -http=:8081 profiles/mem.prof"
