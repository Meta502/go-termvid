package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
	"os/exec"
	"time"
)

import (
	"github.com/AlexEidt/Vidio"
	"github.com/nfnt/resize"
	"golang.org/x/crypto/ssh/terminal"
)

func playAudio(file string) {
	cmd := exec.Command("/usr/bin/mpv", "--no-video", file)
	if err := cmd.Run(); err != nil {
		return
	}
}

func render(frameBuffer chan image.Image, buf *bufio.Writer) {
	for {
		now := time.Now().UnixMicro()
		frame := <-frameBuffer
		bounds := frame.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				r, g, b, _ := frame.At(x, y).RGBA()
				buf.Write([]byte(fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r/0x101, g/0x101, b/0x101)))
				buf.Write([]byte(" "))
				buf.Write([]byte("\033[0m"))
			}
			buf.Flush()
			buf.Write([]byte("\n"))
		}

		buf.WriteString(fmt.Sprintf("\nLast Frame Time: %.2fms", float64(time.Now().UnixMicro()-now)/1000))
		buf.Write([]byte("\033[0;0H"))
	}
}

func decodeFrame(video *vidio.Video, frameBuffer chan image.Image, decoding chan bool, th int) {
	spf := 1.0 / float64(video.FPS())
	ticker := time.NewTicker(time.Duration(spf * float64(time.Second)))

	frame := image.NewRGBA(image.Rect(0, 0, video.Width(), video.Height()))
	video.SetFrameBuffer(frame.Pix)

	for video.Read() {
		<-ticker.C

		width, height, err := terminal.GetSize(th)
		if err != nil {
			fmt.Println(err)
			return
		}

		if err != nil {
			fmt.Println(err)
			return
		}

		frameBuffer <- resize.Resize(uint(width), uint(height-2), frame, resize.Bicubic)
	}
	close(frameBuffer)
	close(decoding)
}

func main() {
	tFd := int(os.Stdout.Fd())

	flagPtr := flag.String("video", "", "video file to play")
	flag.Parse()

	file := *flagPtr

	if file == "" {
		fmt.Println("video file not specified.")
		return
	}

	video, err := vidio.NewVideo(file)

	if err != nil {
		fmt.Println(err)
		return
	}

	running := make(chan bool)
	frameBuffer := make(chan image.Image)

	bufStdout := bufio.NewWriterSize(os.Stdout, 1920*4)

	// Start render thread
	go render(frameBuffer, bufStdout)

	// Start frame and audio output threads
	go decodeFrame(video, frameBuffer, running, tFd)
	go playAudio(file)

	// Halt until channel is closed
	<-running
}
