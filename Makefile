text_downloader: text_downloader.go
	docker run --rm -v $(CURDIR):/usr/src/text_downloader -w /usr/src/text_downloader golang:1.13-alpine go build -v

run: text_downloader
	docker run -it -v $(CURDIR):/usr/src/text_downloader -w /usr/src/text_downloader golang:1.13-alpine ./text_downloader
