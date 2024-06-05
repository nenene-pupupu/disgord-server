# disGOrd

## How to Run
1. check if `go` is installed
    ```sh
    go version
    ```

1. execute shell script `init.sh`, which will create dummy data for test.
    ```sh
    chmod +x init.sh
    ./init.sh
    ```

1. run server
    ```sh
    go build && ./disgord

    # or
    go run main.go
    ```

1. access through `localhost:8080`

1. check APIs at `/swagger/index.html`

1. after each git pull, run `go generate ./...`
