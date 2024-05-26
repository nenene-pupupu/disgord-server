# disGOrd

## How to Run
1. check if `go` is installed
    ```
    go version
    ```

1. execute shell script `init.sh`, which will create dummy data for test.
    ```
    chmod +x init.sh
    ./init.sh
    ```

1. run server
    ```
    go build && ./disgord

    # or
    go run main.go
    ```

1. access through `localhost:8080`

1. check APIs at `/swagger/index.html`

1. after each git pull, run `go generate ./...`
