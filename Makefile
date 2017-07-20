# Copyright © 2017 Syndesis Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BIN := github-example-app

# This repo's root import path (under GOPATH).
PKG := github.com/jimmidyson/github-example-app

# Where to push the docker image.
REGISTRY ?= jimmidyson

# Which architecture to build - see $(ALL_ARCH) for options.
ARCH ?= amd64

# This version-strategy uses git tags to set the version string
BUILD_DATE := $(shell date -u)
VERSION ?= $(shell git describe --match 'v[0-9]*' --dirty --always)

OPENSHIFT_IMAGE_STREAM ?= github-example-app:deploy
OPENSHIFT_CLIENT_IMAGE ?= openshift/origin:v1.5.1

#
# This version-strategy uses a manual value to set the version string
#VERSION := 1.2.3

###
### These variables should not need tweaking.
###

ALL_ARCH := amd64 arm arm64

# Set default base image dynamically for each arch
ifeq ($(ARCH),amd64)
    BASEIMAGE?=alpine:3.6
endif
ifeq ($(ARCH),arm)
    BASEIMAGE?=armhf/alpine:3.6
endif
ifeq ($(ARCH),arm64)
    BASEIMAGE?=owlab/alpine-arm64:3.6
endif

IMAGE := $(REGISTRY)/$(BIN)

GOVERSION ?= 1.8.3
BUILD_IMAGE ?= golang:$(GOVERSION)-alpine
GOLANG_IMAGE ?= golang:$(GOVERSION)

# If you want to build all binaries, see the 'all-build' rule.
# If you want to build all images, see the 'all-image' rule.
# If you want to build AND push all images, see the 'all-push' rule.
all: build

build-%:
	@$(MAKE) --no-print-directory ARCH=$* build

image-%:
	@$(MAKE) --no-print-directory ARCH=$* image

push-%:
	@$(MAKE) --no-print-directory ARCH=$* push

all-build: $(addprefix build-, $(ALL_ARCH))

all-image: $(addprefix image-, $(ALL_ARCH))

all-push: $(addprefix push-, $(ALL_ARCH))

build: bin/$(ARCH)/$(BIN)

bin/$(ARCH)/$(BIN): vendor build-dirs
	@echo "building: $@"
	@docker run                                                              \
	    -ti                                                                  \
	    -u $$(id -u):$$(id -g)                                               \
	    -v $$(pwd)/.go:/go:Z                                                 \
	    -v $$(pwd):/go/src/$(PKG):Z                                          \
	    -v $$(pwd)/bin/$(ARCH):/go/bin:Z                                     \
	    -v $$(pwd)/bin/$(ARCH):/go/bin/linux_$(ARCH):Z                       \
	    -v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static:Z  \
	    -w /go/src/$(PKG)                                                    \
	    $(BUILD_IMAGE)                                                       \
	    /bin/sh -c "                                                         \
	        ARCH=$(ARCH)                                                     \
	        VERSION=$(VERSION)                                               \
			BUILD_DATE=\"$(BUILD_DATE)\"                                     \
	        PKG=$(PKG)                                                       \
	        ./build/build.sh                                                 \
	    "

vendor: build-dirs .vendor Gopkg.lock Gopkg.toml
.vendor:
	@echo "updating vendored deps"
	@docker run                                                              \
	    -ti                                                                  \
	    -v $$(pwd):/go/src/$(PKG):Z                                         \
	    -v $$(pwd)/bin/$(ARCH):/go/bin:Z                                     \
	    -w /go/src/$(PKG)                                                    \
	    $(GOLANG_IMAGE)                                                     \
	    /bin/sh -c "                                                         \
	        ./build/update_vendor.sh                                         \
	    "
	@touch .vendor

DOTFILE_IMAGE = $(subst :,_,$(subst /,_,$(IMAGE))-$(VERSION))

image: .image-$(DOTFILE_IMAGE) image-name
.image-$(DOTFILE_IMAGE): bin/$(ARCH)/$(BIN) Dockerfile.in
	@sed \
	    -e 's|ARG_BIN|$(BIN)|g' \
	    -e 's|ARG_ARCH|$(ARCH)|g' \
	    -e 's|ARG_FROM|$(BASEIMAGE)|g' \
	    Dockerfile.in > .dockerfile-$(ARCH)
	@docker build -t $(IMAGE):$(VERSION) -f .dockerfile-$(ARCH) .
	@docker images -q $(IMAGE):$(VERSION) > $@

image-name:
	@echo "image: $(IMAGE):$(VERSION)"

push: .push-$(DOTFILE_IMAGE) push-name
.push-$(DOTFILE_IMAGE): .image-$(DOTFILE_IMAGE)
ifeq ($(findstring gcr.io,$(REGISTRY)),gcr.io)
	@gcloud docker -- push $(IMAGE):$(VERSION)
else
	@docker push $(IMAGE):$(VERSION)
endif
	@docker images -q $(IMAGE):$(VERSION) > $@

push-name:
	@echo "pushed: $(IMAGE):$(VERSION)"

circleci-deploy: push
	@docker run --entrypoint=/bin/sh $(OPENSHIFT_CLIENT_IMAGE) -c "                      \
	  openshift cli login $$OPENSHIFT_SERVER --token=$$OPENSHIFT_TOKEN &&                \
	  openshift cli project $$OPENSHIFT_PROJECT &&                                       \
	  openshift cli tag --source=docker $(IMAGE):$(VERSION) $(OPENSHIFT_IMAGE_STREAM) && \
	  openshift cli import-image $(OPENSHIFT_IMAGE_STREAM)                               \
	"

version:
	@echo $(VERSION)

test: build-dirs
	@docker run                                                              \
	    -ti                                                                  \
	    -u $$(id -u):$$(id -g)                                               \
	    -v $$(pwd)/.go:/go:Z                                                 \
	    -v $$(pwd):/go/src/$(PKG):Z                                          \
	    -v $$(pwd)/bin/$(ARCH):/go/bin:Z                                     \
	    -v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static:Z  \
	    -w /go/src/$(PKG)                                                    \
	    $(BUILD_IMAGE)                                                       \
	    /bin/sh -c "                                                         \
	        ./build/test.sh                                                  \
	    "

build-dirs:
	@mkdir -p bin/$(ARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(ARCH)

clean: image-clean bin-clean vendor-clean

image-clean:
	rm -rf .image-* .dockerfile-* .push-*

bin-clean:
	rm -rf .go bin

vendor-clean:
	rm -f .vendor
