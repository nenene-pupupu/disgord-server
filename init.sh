#!/bin/bash

if ! command -v go &> /dev/null; then
    echo "'go' command not found. please install 'go'."
    exit 1
fi

go generate ./...
go mod tidy

if [ ! -f ".env" ]; then
    echo "SECRET=$(openssl rand -hex 8)" > .env
    echo "new .env file created."
fi

go build

./disgord &
sleep 2

for i in {1..6}; do
    curl -X POST \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"user$i\",\"password\":\"user$i\"}" \
        http://localhost:8080/auth/sign-up
    echo
    sleep 1
done

pkill disgord
