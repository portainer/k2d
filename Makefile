# go-workbench Makefile template v1.1.0
# For a list of valid GOOS and GOARCH values, see: https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63
# Note: these can be overriden on the command line e.g. `make PLATFORM=<platform> ARCH=<arch>`
PLATFORM="linux"
ARCH="$(shell go env GOARCH)"
ARM=""
VERSION="latest"

.PHONY: pre dev build release image image-arm-6 image-arm-7 image-multiarch clean reset

dist := dist
bin := $(shell basename $(CURDIR))
image := portainer/k2d:$(VERSION)

pre:
	mkdir -pv $(dist) 

dev: pre
	air -c .air.toml

build: pre
	GOOS=$(PLATFORM) GOARCH=$(ARCH) GOARM=$(ARM) CGO_ENABLED=0 go build --installsuffix cgo --ldflags '-s' -o $(bin) cmd/k2d.go
	mv $(bin) $(dist)/

release: pre
	GOOS=$(PLATFORM) GOARCH=$(ARCH) GOARM=$(ARM) CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags '-s' -o $(bin) cmd/k2d.go
	mv $(bin) $(dist)/

image: release
	docker buildx build --push --platform=$(PLATFORM)/$(ARCH) -t $(image)-$(PLATFORM)-$(ARCH) .

image-arm-6:
	$(MAKE) release PLATFORM=linux ARCH=arm ARM=6
	docker buildx build --push --platform=linux/arm/v6 -t $(image)-linux-armv6 .

image-arm-7: 
	$(MAKE) release PLATFORM=linux ARCH=arm ARM=7
	docker buildx build --push --platform=linux/arm/v7 -t $(image)-linux-armv7 .

image-multiarch:
	$(MAKE) image PLATFORM=linux ARCH=amd64
	$(MAKE) image PLATFORM=linux ARCH=386
	$(MAKE) image PLATFORM=linux ARCH=arm64
	$(MAKE) image-arm-7
	$(MAKE) image-arm-6
	docker buildx imagetools create -t $(image) \
		$(image)-linux-amd64 \
		$(image)-linux-arm64 \
		$(image)-linux-386 \
		$(image)-linux-armv6 \
		$(image)-linux-armv7

clean:
	rm -rf $(dist)/*
	rm -rf /opt/dev-toolkit/k2d/*
	rm -rf /var/lib/k2d

reset: build
	$(dist)/$(bin) -reset