CC := gcc
FLAGS := -lX11 -lXext -O3
version=n7.0
srcPath=tmp/$(version)/src
patchPath=$(CURDIR)/patches/ffmpeg/$(version)
CGO_CFLAGS := -I$(CURDIR)/gamecapture -I$(CURDIR)/tmp/$(version)/include/
CGO_LDFLAGS := -L$(CURDIR)/gamecapture -L$(CURDIR)/tmp/$(version)/lib/
PKG_CONFIG_PATH := $(CURDIR)/tmp/$(version)/lib/pkgconfig
configure := --enable-libx264 --enable-libx265 --enable-decoder=hevc --enable-gpl --enable-nonfree --enable-nvenc
configure-client-only := --enable-libx264 --enable-libx265 --enable-libaom --enable-gpl --enable-nonfree
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	CGO_CFLAGS += -I/usr/local/cuda/include
	CGO_LDFLAGS += -L/usr/local/cuda/lib64
else ifeq ($(UNAME_S),Darwin)
	CGO_LDFLAGS += -ld_classic
else ifeq ($(findstring MINGW,$(UNAME_S)),MINGW)
endif

all: vaporplay vaporplay-native-client

vaporplay-native-client: $(srcPath)
	go mod tidy && cd client/vaporplay-native-client && go mod tidy
	cd ./client/vaporplay-native-client && PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -ldflags "-s -w" -o ../../vaporplay-native-client

vaporplay: server/webui gamecapture/libwindowmatch.so gamecapture/libgamecapture.so $(srcPath)
	go mod tidy && cd server && go mod tidy
	cd ./server && PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -ldflags "-s -w" -o ../vaporplay

gamecapture/libwindowmatch.so: gamecapture/window_match.c gamecapture/window_match.h
	cd gamecapture && $(CC) -shared -o libwindowmatch.so -fPIC window_match.c $(FLAGS)

gamecapture/libgamecapture.so: gamecapture/game_capture.c gamecapture/game_capture.h
	cd gamecapture && $(CC) -shared -o libgamecapture.so -fPIC game_capture.c $(FLAGS)

server/webui: client/vaporplay-web-client
	cd client/vaporplay-web-client && npm i && npm run build
	rm -rf server/webui
	cp -r client/vaporplay-web-client/dist server/webui

$(srcPath):
	rm -rf $(srcPath)
	mkdir -p $(srcPath)
	cd $(srcPath) && git clone --branch $(version) https://github.com/FFmpeg/FFmpeg .
	# apply patches
	cd $(srcPath) && \
		for patch in $(wildcard $(patchPath)/*.patch); do \
			echo "Applying patch $${patch}"; \
			patch -p1 < $${patch}; \
		done
	cd $(srcPath) && ./configure --prefix=.. $(configure)
	cd $(srcPath) && make -j8
	cd $(srcPath) && make install

build-client-only-ffmpeg:
	cd $(srcPath) && ./configure --prefix=.. $(configure-client-only)
	cd $(srcPath) && make -j8
	cd $(srcPath) && make install

clean:
	rm -f gamecapture/*.so vaporplay vaporplay-native-client
	rm -rf ./server/webui/
	go clean -cache

clean-deps: clean
	rm -rf ./client/vaporplay-web-client/node_modules/
	rm -rf $(srcPath)

.PHONY: clean clean-deps vaporplay build-client-only-ffmpeg vaporplay-native-client all
