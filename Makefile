all: clean crtsh-web

crtsh-web:
	CGO_ENABLED=0 go build -o $@ -ldflags "-X github.com/crtsh/crtsh-web/config.BuildTimestamp=`date --utc +%Y-%m-%dT%H:%M:%SZ`"

clean:
	rm -f crtsh-web
