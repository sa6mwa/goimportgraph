BIN=goimportgraph
DEST=/usr/local/bin

.PHONY: all build install clean

all: build

build: $(BIN)

$(BIN):
	go build -o $(BIN) -ldflags=-s .

install: 
	install $(BIN) $(DEST)/$(BIN)

clean:
	rm -f $(BIN)
