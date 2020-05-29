TARGET = ios_video_pull

all: $(TARGET)

$(TARGET): main.go go.sum nanowriter.go
	go build -o $(TARGET) .

go.sum:
	go get
	go get .

clean:
	$(RM) $(TARGET) go.sum
