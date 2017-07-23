all:
	go build

um:
	sudo umount test

run:
	./go test test		