TARGET = ios_video_pull

all: $(TARGET)

$(TARGET): main.go go.sum nanowriter.go vendor/github.com/danielpaulus/quicktime_video_hack/screencapture/usbadapter.go
	go build  -o $(TARGET) .

go.sum:
	go get
	go get .

clean:
	$(RM) $(TARGET) go.sum