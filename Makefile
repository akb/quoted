.PHONY: all clean test;

quoted = bin/quoted

all: clean $(quoted)

clean:
	rm -f $(quoted)
	mkdir -p bin

$(quoted):
	go build -o $(quoted) ./quoted/main.go ./quoted/logger.go ./quoted/quote.go

test:
	ruby test.rb
