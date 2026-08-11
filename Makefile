all: clean generate crtsh-web

generate:
	go run github.com/valyala/quicktemplate/qtc@latest -dir=request/templates

crtsh-web:
	CGO_ENABLED=0 go build -o $@ -ldflags "-X github.com/crtsh/crtsh-web/config.BuildTimestamp=`date --utc +%Y-%m-%dT%H:%M:%SZ`"

clean:
	rm -f crtsh-web request/templates/*.qtpl.go
