GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

PKG_DIR = $(CURDIR)/package-$(GOOS)-$(GOARCH)
TOOLS_DIR = $(PKG_DIR)/tools
# NOTE: variable GIT_HASH should not contain spaces or quotes or build will fail
GIT_HASH = $(shell git describe --tags)
ARC_FILE = package-$(GOOS)-$(GOARCH)-$(GIT_HASH).tar.gz
ARC_LITE = lite-$(GOOS)-$(GOARCH)-$(GIT_HASH).tar.gz

ifeq ($(GOOS), windows)
GOEXT := .exe
else
GOEXT :=
endif

Rhoc_EXE = $(PKG_DIR)/Rhoc$(GOEXT)
PACKER_EXE = $(TOOLS_DIR)/packer$(GOEXT)
TERRAFORM_EXE = $(TOOLS_DIR)/terraform$(GOEXT)

$(PACKER_EXE):
	@echo Building packer
	@git submodule update --init
	@mkdir -p $(TOOLS_DIR)
	@cd hashicorp/packer && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod=vendor -o $(PACKER_EXE)
	@cd hashicorp/packer && git reset --hard

$(TERRAFORM_EXE):
	@echo Building terraform
	@git submodule update --init
	@mkdir -p $(TOOLS_DIR)
	@cd hashicorp/terraform && GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod=vendor -o $(TERRAFORM_EXE)
	@cd hashicorp/terraform && git reset --hard

bin:
	@echo Building Rhoc
	@mkdir -p $(PKG_DIR)
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-X Rhoc/cmd.VERSION=$(GIT_HASH)" -mod=vendor -o $(Rhoc_EXE)

package: bin $(PACKER_EXE) $(TERRAFORM_EXE)
	@echo Preparing package
	@rm -rf $(PKG_DIR)/distrib/ $(PKG_DIR)/postprocess/ $(PKG_DIR)/templates/ $(PKG_DIR)/examples/
	@cp -R $(CURDIR)/distrib/ $(CURDIR)/postprocess/ $(CURDIR)/templates/ $(CURDIR)/examples/ $(PKG_DIR)

$(ARC_FILE): package
	@echo Making package $(ARC_FILE)
	@cd $(PKG_DIR) && tar -czf ../$(ARC_FILE) *

$(ARC_LITE): package
	@echo Making lite package $(ARC_LITE)
	@cd $(PKG_DIR) && tar -czf ../$(ARC_LITE) templates/ postprocess/ examples/ Rhoc

all: $(ARC_FILE)

lite: $(ARC_LITE)

.PHONY: bin package all lite
.DEFAULT_GOAL := all

