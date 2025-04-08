CC := gcc
FLAGS := -lX11 -lXext -O3
version=n7.0
srcPath=tmp/$(version)/src
CGO_CFLAGS := -I$(CURDIR)/gamecapture -I$(CURDIR)/tmp/$(version)/include/ -I/usr/local/cuda/include
CGO_LDFLAGS := -L$(CURDIR)/gamecapture -L$(CURDIR)/tmp/$(version)/lib/ -L/usr/local/cuda/lib64
PKG_CONFIG_PATH := $(CURDIR)/tmp/$(version)/lib/pkgconfig
configure := --enable-libx264 --enable-gpl --enable-nonfree --enable-nvenc

vaporplay: tmp/webui gamecapture/libwindowmatch.so gamecapture/libgamecapture.so
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -ldflags "-s -w" -o vaporplay

gamecapture/libwindowmatch.so: gamecapture/window_match.c gamecapture/window_match.h
	cd gamecapture && $(CC) -shared -o libwindowmatch.so -fPIC window_match.c $(FLAGS)

gamecapture/libgamecapture.so: gamecapture/game_capture.c gamecapture/game_capture.h
	cd gamecapture && $(CC) -shared -o libgamecapture.so -fPIC game_capture.c $(FLAGS)

tmp/webui: www/vaporplay-client
	cd www/vaporplay-client && npm run build
	rm -rf tmp/webui
	cp -r www/vaporplay-client/dist tmp/webui

install-ffmpeg:
	rm -rf $(srcPath)
	mkdir -p $(srcPath)
	cd $(srcPath) && git clone https://github.com/FFmpeg/FFmpeg .
	cd $(srcPath) && git checkout $(version)
	cd $(srcPath) && ./configure --prefix=.. $(configure)
	cd $(srcPath) && make -j8
	cd $(srcPath) && make install

clean:
	rm -f gamecapture/*.so vaporplay
	go clean -cache

.PHONY: clean vaporplay install-ffmpeg
