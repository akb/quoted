.PHONY: all clean run test;

quoted = bin/quoted

all: clean $(quoted) run

clean:
	rm -f $(quoted)
	mkdir -p bin

$(quoted):
	go build -o $(quoted) ./quoted/main.go ./quoted/logger.go ./quoted/quote.go

run:
	bin/quoted

test:
	ruby test.rb
