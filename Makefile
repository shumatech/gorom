.DEFAULT_GOAL := all
APPS=gorom
gorom_DIR=cli
gorom_SRCS=main.go fixrom.go chkrom.go chktor.go dir2dat.go lstor.go fltdat.go fuzzymv.go torzip.go goromdb.go
goromui_DIR=gui
goromui_SRCS=main.go chkrom.go fixrom.go
common_SRCS=romdb/romdb.go dat/dat.go util/util.go romio/romio.go torrent/torrent.go checksum/checksum.go term/term.go torzip/torzip.go

BINDIR=bin
RESDIR=res
INSTALLDIR=install
MKDIR=mkdir -p
VERSION?=$(shell git describe --tags --always)
BUILD:=$(shell date +%FT%T%z)
Q?=@
OS:=$(shell uname -s | cut -c -7 | tr [A-Z] [a-z])

#
# Windows
#
ifeq ($(OS),mingw32)

APPS+=goromui
EXE=.exe
export CGO_LDFLAGS=-Wl,--enable-stdcall-fixup
gorom_EXTLDFLAGS=-static
goromui_EXTLDFLAGS=-Wl,-subsystem,windows -static res/icon.o res/gorom.o
SCITER=sciter.dll
REVISION:=$(shell git rev-list --count --first-parent HEAD)
ARCHIVE=gorom-windows-$(VERSION).zip

bin/goromui.exe: res/icon.o res/gorom.o

res/icon.o: res/icon.rc res/gorom.ico
	$(Q)windres $^ $@

res/gorom.rc.out: res/gorom.rc
	$(Q)sed -e "s/%VERSION%/$(VERSION)/" -e "s/%REVISION%/$(REVISION)/" $^ > $@

res/gorom.o: res/gorom.rc.out
	$(Q)windres $^ $@

install:
	@echo INSTALL
	$(Q)cd $(BINDIR); \
	zip $(ARCHIVE) $(foreach app,$(APPS),$(app)$(EXE)) $(SCITER)

#
# Linux
#
else ifeq ($(OS),linux)

APPS+=goromui
SCITER=libsciter-gtk.so
ARCHIVE=gorom-linux-$(VERSION).tgz

install:
	@echo INSTALL
	$(Q)tar -C $(BINDIR) -cvzf $(ARCHIVE) $(foreach app,$(APPS),$(app)$(EXE)) $(SCITER)

#
# OS X
#
else ifeq ($(OS),darwin)

APPS+=goromui
SCITER=sciter-osx-64.dylib
APP=GoROM.app
DMG=gorom-osx-$(VERSION).dmg
VOLUME=GoROM
BACKGROUND=$(INSTALLDIR)/background.png

app: goromui
	mkdir -p $(BINDIR)/$(APP)/Contents/MacOS
	mkdir -p $(BINDIR)/$(APP)/Contents/Resources
	cp -f $(INSTALLDIR)/Info.plist $(BINDIR)/$(APP)/Contents
	echo -n "APPL????" > $(BINDIR)/$(APP)/Contents/PkgInfo
	ln -f $(BINDIR)/goromui $(BINDIR)/$(APP)/Contents/MacOS/goromui
	ln -f $(BINDIR)/$(SCITER) $(BINDIR)/$(APP)/Contents/MacOS/$(SCITER)
	cp -f $(RESDIR)/gorom.icns $(BINDIR)/$(APP)/Contents/Resources

app_clean:
	@echo CLEAN APP
	$(Q)rm -rf $(BINDIR)/$(APP)

dmg: app gorom
	hdiutil create -ov -megabytes 25 -fs HFS+ -volname $(VOLUME) $(BINDIR)/$(DMG)
	hdiutil attach -noautoopen $(BINDIR)/$(DMG)
	cp -R $(BINDIR)/$(APP) /Volumes/$(VOLUME)/
	cp $(BINDIR)/gorom$(EXE) /Volumes/$(VOLUME)/
	ln -s /Applications /Volumes/$(VOLUME)/Applications
	ln -s /usr/local/bin /Volumes/$(VOLUME)/bin
	mkdir /Volumes/$(VOLUME)/.background
	cp $(BACKGROUND) /Volumes/$(VOLUME)/.background
	osascript < $(INSTALLDIR)/dmgwin.osa
	hdiutil detach /Volumes/$(VOLUME)/
	hdiutil convert -format UDBZ -o $(BINDIR)/tmp$(DMG) $(BINDIR)/$(DMG)
	mv -f $(BINDIR)/tmp$(DMG) $(BINDIR)/$(DMG)

dmg_clean:
	@echo CLEAN DMG
	$(Q)rm -f $(BINDIR)/$(DMG)


clean: app_clean dmg_clean
install: dmg

endif

DEBUG ?= 0
ifeq ($(DEBUG),1)
	TAGS=-tags debug
else
	goromui_SRCS+=res.go 
endif

gui/res.go: $(wildcard gui/res/*)
	@echo PACK $@
	$(Q)packfolder gui/res $@ -go

define target
$(BINDIR)/$(1)$(EXE): $$(foreach src,$$($(1)_SRCS),$(2)/$$(src)) $(common_SRCS) | $(BINDIR)
	@echo GO $(1)
	$(Q)go build -o $(BINDIR)/$(1)$(EXE) $(TAGS) -ldflags "-X main.Version=$(VERSION) -X main.Build=$(BUILD) -extldflags '$($(1)_EXTLDFLAGS)'" -trimpath gorom/$(2)
	$(Q)strip $$@
$(1): $(BINDIR)/$(1)$(EXE)
$(1)_clean: 
	@echo CLEAN $(1)
	$(Q)rm -f $(BINDIR)/$(1)$(EXE)
$(1)_test: 
	$(Q)go test -v -cover gorom/$(2)
$(1)_upx: $(BINDIR)/$(1)$(EXE)
	@echo PACK $$<
	$(Q)upx -qqt $$< || upx $$<
endef

$(foreach app,$(APPS),$(eval $(call target,$(app),$($(app)_DIR))))

$(BINDIR):
	$(Q)$(MKDIR) -p $(BINDIR)

all: $(foreach app,$(APPS),$(app))

clean: $(foreach app,$(APPS),$(app)_clean)

upx: $(foreach app,$(APPS),$(app)_upx)

test:
	$(Q)go test -v ./...

cover:
	$(Q)go test -coverprofile=coverage.out -v ./...
	$(Q)go tool cover -html=coverage.out

.PHONY: all clean upx test cover install
