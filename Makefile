PREFIX?=/usr/local
MODE?=docker
MODULES?=remote-object
SRC_FILES:=$(shell find baetyl-remote-object -type f -name '*.go')
PLATFORM_ALL:=darwin/amd64 linux/amd64 linux/arm64 linux/386 linux/arm/v7 linux/arm/v6 linux/arm/v5 linux/ppc64le linux/s390x

GIT_REV:=git-$(shell git rev-parse --short HEAD)
GIT_TAG:=$(shell git tag --contains HEAD)
VERSION:=$(if $(GIT_TAG),$(GIT_TAG),$(GIT_REV))
# CHANGES:=$(if $(shell git status -s),true,false)

GO_OS:=$(shell go env GOOS)
GO_ARCH:=$(shell go env GOARCH)
GO_ARM:=$(shell go env GOARM)
GO_FLAGS?=-ldflags "-X 'github.com/baetyl/baetyl/cmd.Revision=$(GIT_REV)' -X 'github.com/baetyl/baetyl/cmd.Version=$(VERSION)'"
GO_TEST_FLAGS?=
GO_TEST_PKGS?=$(shell go list ./...)

ifndef PLATFORMS
	GO_OS:=$(shell go env GOOS)
	GO_ARCH:=$(shell go env GOARCH)
	GO_ARM:=$(shell go env GOARM)
	PLATFORMS:=$(if $(GO_ARM),$(GO_OS)/$(GO_ARCH)/$(GO_ARM),$(GO_OS)/$(GO_ARCH))
	ifeq ($(GO_OS),darwin)
		PLATFORMS+=linux/amd64
	endif
else ifeq ($(PLATFORMS),all)
	override PLATFORMS:=$(PLATFORM_ALL)
endif

OUTPUT:=output
OUTPUT_DIRS:=$(PLATFORMS:%=$(OUTPUT)/%/baetyl)
OUTPUT_BINS:=$(OUTPUT_DIRS:%=%/bin/baetyl)
OUTPUT_PKGS:=$(OUTPUT_DIRS:%=%/baetyl-$(VERSION).zip) # TODO: switch to tar

OUTPUT_MODS:=$(MODULES:%=baetyl-%)
IMAGE_MODS:=$(MODULES:%=image/baetyl-%) # a little tricky to add prefix 'image/' in order to distinguish from OUTPUT_MODS
NATIVE_MODS:=$(MODULES:%=native/baetyl-%) # a little tricky to add prefix 'native/' in order to distinguish from OUTPUT_MODS

.PHONY: all $(OUTPUT_MODS)
all: baetyl $(OUTPUT_MODS)

baetyl: $(OUTPUT_BINS) $(OUTPUT_PKGS)

$(OUTPUT_BINS): $(SRC_FILES)
	@echo "BUILD $@"
	@mkdir -p $(dir $@)
	@# baetyl failed to collect cpu related data on darwin if set 'CGO_ENABLED=0' in compilation
	@$(shell echo $(@:$(OUTPUT)/%/baetyl/bin/baetyl=%)  | sed 's:/v:/:g' | awk -F '/' '{print "GOOS="$$1" GOARCH="$$2" GOARM="$$3" go build"}') -o $@ ${GO_FLAGS} .

$(OUTPUT_MODS):
	@make -C $@

.PHONY: image $(IMAGE_MODS)
image: $(IMAGE_MODS) 

$(IMAGE_MODS):
	@make -C $(notdir $@) image

.PHONY: rebuild
rebuild: clean all

.PHONY: clean
clean:
	@-rm -rf $(OUTPUT)

.PHONY: fmt
fmt:
	go fmt  ./...