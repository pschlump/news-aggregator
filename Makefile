
all:
	go build

test:
	( cd index ; go test )
	( cd unzip ; go test )
	( cd naLib ; go test )



