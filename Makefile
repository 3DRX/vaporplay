CC := gcc
FLAGS := -lX11 -lXext
version=n7.0
srcPath=tmp/$(version)/src
CGO_CFLAGS := -I$(CURDIR)/gamecapture -I$(CURDIR)/tmp/$(version)/include/
CGO_LDFLAGS := -L$(CURDIR)/gamecapture -L$(CURDIR)/tmp/$(version)/lib/
PKG_CONFIG_PATH := $(CURDIR)/tmp/$(version)/lib/pkgconfig
patchPath=
configure=

piongs: gamecapture/libwindowmatch.so gamecapture/libgamecapture.so
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -o piongs

gamecapture/libwindowmatch.so: gamecapture/window_match.c gamecapture/window_match.h
	cd gamecapture && $(CC) -shared -o libwindowmatch.so -fPIC window_match.c $(FLAGS)

gamecapture/libgamecapture.so: gamecapture/game_capture.c gamecapture/game_capture.h
	cd gamecapture && $(CC) -shared -o libgamecapture.so -fPIC game_capture.c $(FLAGS)

install-ffmpeg:
	rm -rf $(srcPath)
	mkdir -p $(srcPath)
	cd $(srcPath) && git clone https://github.com/FFmpeg/FFmpeg .
	cd $(srcPath) && git checkout $(version)
ifneq "" "$(patchPath)"
	cd $(srcPath) && git apply $(patchPath)
endif
	cd $(srcPath) && ./configure --prefix=.. $(configure)
	cd $(srcPath) && make -j8
	cd $(srcPath) && make install

clean:
	rm -f gamecapture/*.so piongs
	go clean -cache

.PHONY: clean piongs install-ffmpeg
