run: bin/theunwrapper
	@@./bin/theunwrapper

bin/theunwrapper:
	go build -o bin/theunwrapper .

clean:
	rm -rf bin/*

.PHONY: docker
docker:
	docker buildx build --platform linux/arm64,linux/amd64 -f Dockerfile -t ${REGISTRY}/theunwrapper --push .
