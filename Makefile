all:
	go build

filter:	all

install:	mondrian
	cp $< ${HOME}/bin
