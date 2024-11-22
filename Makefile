
.PHONY: build push install run stop clean

install:
	./install.sh

build:
	./build.sh

push:
	./push.sh

run:
	docker-compose up -d

stop:
	docker-compose down

clean:
	docker-compose down --rmi all