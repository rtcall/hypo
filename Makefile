all: hypo hypoc

hypo: $(wildcard cmd/hypo/*.go) $(wildcard cpu/*.go)
	go build ./cmd/hypo

hypoc: $(wildcard cmd/hypoc/*.go) $(wildcard asm/*.go)
	go build ./cmd/hypoc

clean:
	rm -f hypo hypoc
