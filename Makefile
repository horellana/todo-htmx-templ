dev-templ:
		PORT=6000 templ generate --watch --proxy="http://localhost:6000" --cmd="air"

dev-tailwindcss:
		tailwindcss -i input.css -o static/output.css --watch

build-templ:
		templ generate -f templates

build-tailwindcss:
		tailwindcss -i input.css -o static/output.css

build-gocode:
		go get
		go build .

build:
		$(MAKE) build-templ
		$(MAKE) build-tailwindcss
		$(MAKE) build-gocode
