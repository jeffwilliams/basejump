archs=386 amd64 arm arm64
files=$(foreach arch, $(archs), plugin-$(arch)/basejump)


all: $(files)

plugin-%/basejump: 
	GOARCH=$* go build -o $@

clean:
	rm -r $(files)
