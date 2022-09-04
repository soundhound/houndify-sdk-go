.PHONY: run
run: main
	./$<

main: *.go go.mod
	go build -o $@ .

.PHONY: all
all: main