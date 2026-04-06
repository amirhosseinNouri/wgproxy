BINARY  = wg0proxy
GOFLAGS = -ldflags="-s -w"

.PHONY: build clean

build:
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(BINARY)

clean:
	rm -f $(BINARY)
