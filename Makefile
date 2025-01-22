CC := gcc
FLAGS := -L/usr/lib/x86_64-linux-gnu -lX11 -lXext

SOURCES := $(wildcard ./*.c)
OBJECTS := $(SOURCES:./%.c=./%.o)

piongs: libwindow-match.so
	go build -o piongs

libwindow-match.so: window_match.c window_match.h
	$(CC) -shared -fPIC window_match.c -o libwindow-match.so $(FLAGS)

main: $(OBJECTS)
	$(CC) $(OBJECTS) -o main $(FLAGS)

%.o: %.c
	$(CC) -c $< -o $@

clean:
	rm -f $(OBJECTS) *.so main

.PHONY: clean
