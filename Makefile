COMMIT_HASH = $(shell git rev-parse --short HEAD)
BRANCH_NAME = $(shell git rev-parse --abbrev-ref HEAD)
RET = $(shell git describe --contains $(COMMIT_HASH) 1>&2 2> /dev/null; echo $$?)
USER = $(shell whoami)
IS_DIRTY_CONDITION = $(shell git diff-index --name-status HEAD | wc -l)

REPO = ghcr.io/nchc-ai
IMAGE = $(notdir $(CURDIR))

ifeq ($(strip $(IS_DIRTY_CONDITION)), 0)
	# if clean,  IS_DIRTY tag is not required
	IS_DIRTY = $(shell echo "")
else
	# add dirty tag if repo is modified
	IS_DIRTY = $(shell echo "-dirty")
endif

# Image Tag rule
# 1. Use git tag if we found
# 2. if repo in a master branch, use <username>-<commit-hash>
# 3. if repo in non-master branch, use branch name as image tag
ifeq ($(RET),0)
	TAG = $(shell git describe --contains $(COMMIT_HASH))$(IS_DIRTY)
else
	ifeq ($(BRANCH_NAME), main)
		TAG = $(USER)-$(COMMIT_HASH)$(IS_DIRTY)
	else
		TAG = $(BRANCH_NAME)$(IS_DIRTY)
	endif
endif

install_deps:
	go mod download

run-backend:
	rm -rf bin/*
	go mod vendor
	go build -mod=vendor -o bin/app .
	./bin/app --logtostderr=true --conf=./conf/api-config-dev.json

run-backend-docker:
	docker run -ti --rm  -p 38080:38080 $(REPO)/$(IMAGE):$(TAG)

image:
	docker build -t $(REPO)/$(IMAGE):$(TAG) .


# https://github.com/swaggo/swag/issues/532#issuecomment-539461837
# swag v1.6.3 is not compatible older version
# go get -u github.com/swaggo/swag/cmd/swag@v1.6.2
build-doc:
	$(GOPATH)/bin/swag init

clean:
	rm -rf bin/*
