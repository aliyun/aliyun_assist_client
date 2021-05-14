all: linux windows freebsd

linux:
	GOOS=linux CGO_ENABLED=0 go build -o bin/aliyunservice-linux main.go patch_linux.go
windows:
	GOOS=windows CGO_ENABLED=0 go build -o bin/aliyunservice.exe main.go patch_windows.go
freebsd:
	GOOS=freebsd CGO_ENABLED=0 go build -o bin/aliyunservice-freebsd main.go patch_freebsd.go