install:
	go install

PLATFORMS := linux/amd64 linux/arm darwin/amd64 windows/amd64

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

release: $(PLATFORMS)

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -ldflags "-w" -o kcount-$(os)-$(arch)
	tar -cf - kcount-$(os)-$(arch) | gzip -9c > kcount-$(os)-$(arch).tar.gz
	rm -f kcount-$(os)-$(arch)