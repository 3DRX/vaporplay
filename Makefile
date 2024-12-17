CC := gcc
FLAGS := -L/usr/lib/x86_64-linux-gnu -lX11 -lXext -lvpx

SOURCES := $(wildcard ./*.c)
OBJECTS := $(SOURCES:./%.c=./%.o)

main: $(OBJECTS)
	$(CC) $(OBJECTS) -o main $(FLAGS)

%.o: %.c
	$(CC) -c $< -o $@

clean:
	rm -f $(OBJECTS) main

.PHONY: clean
