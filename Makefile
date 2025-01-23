CC := gcc
FLAGS := -lX11 -lXext
CGO_CFLAGS := -I$(CURDIR)/gamecapture
CGO_LDFLAGS := -L$(CURDIR)/gamecapture

piongs: gamecapture/libwindowmatch.so gamecapture/libgamecapture.so
	CGO_CFLAGS=$(CGO_CFLAGS) CGO_LDFLAGS=$(CGO_LDFLAGS) go build -o piongs

gamecapture/libwindowmatch.so: gamecapture/window_match.c gamecapture/window_match.h
	cd gamecapture && $(CC) -shared -o libwindowmatch.so -fPIC window_match.c $(FLAGS)

gamecapture/libgamecapture.so: gamecapture/game_capture.c gamecapture/game_capture.h
	cd gamecapture && $(CC) -shared -o libgamecapture.so -fPIC game_capture.c $(FLAGS)

clean:
	rm -f gamecapture/*.so piongs

.PHONY: clean piongs
