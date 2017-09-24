SOURCE=$(shell find . -iname '*.go')
SSLCERTS=$(shell find /etc/ssl/certs/ca-certificates.crt /usr/local/etc/openssl/cert.pem 2>/dev/null)

test: build/httppub.tested

build/httppub.tested: $(SOURCE)
	go test -v ./...
	touch build/httppub.tested

build/httppub: build/httppub.tested Makefile
	GOOS=linux go build -o build/httppub *.go

build/ca-certificates.crt: $(SSLCERTS)
	cat $(SSLCERTS) > build/ca-certificates.crt

build/httppub.docker: build/ca-certificates.crt build/httppub Makefile Dockerfile
	docker build -t httppub -f Dockerfile .
	touch build/httppub.docker

DOCKER_APPNAME=httppub
run: build/httppub.docker
	docker run \
		-d --init \
		-p $(PORT):3000 \
		--name $(DOCKER_APPNAME)-GREEN \
		$(DOCKER_OPTIONS) \
		--restart always \
		-v /tmp:/tmp \
		httppub -targets $(TARGETS)
	(docker rm -f $(DOCKER_APPNAME) || true) >/dev/null
	docker rename $(DOCKER_APPNAME)-GREEN $(DOCKER_APPNAME)
