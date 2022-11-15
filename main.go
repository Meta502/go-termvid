package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
	"os/exec"

	"time"

	"github.com/nfnt/resize"
	"golang.org/x/crypto/ssh/terminal"
)

func playAudio() {
	cmd := exec.Command("/usr/bin/mpv", "--no-video", "videoplayback.mp4")
	if err := cmd.Run(); err != nil {
		return
	}
}

func renderImages(image chan image.Image, buf *bufio.Writer) {
	for {
		now := time.Now().UnixMicro()
		frame := <-image
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
			buf.Write([]byte("\n"))
		}

		buf.WriteString(fmt.Sprintf("\nFrame Time: %.2fms", float64(time.Now().UnixMicro()-now)/1000))

		buf.Write([]byte("\033[0;0H"))
		buf.Flush()
	}
}

func main() {
	tFd := int(os.Stdout.Fd())

	fpsPtr := flag.Float64("fps", 24.0, "frame render FPS")
	flag.Parse()

	fps := *fpsPtr

	d, err := os.ReadDir("./out")
	if err != nil {
		fmt.Println(err)
		return
	}

	fc := len(d)

	rch := make(chan image.Image)

	bufStdout := bufio.NewWriterSize(os.Stdout, 2*(800*600))

	go renderImages(rch, bufStdout)

	spf := 1.0 / float64(fps)
	ticker := time.NewTicker(time.Duration(spf * float64(time.Second)))

	go playAudio()
	for i := 0; i < fc; i++ {
		<-ticker.C
		dat, err := os.ReadFile(fmt.Sprintf("out/image-%d.jpg", i+1))

		if err != nil {
			fmt.Println(err)
			return
		}

		width, height, err := terminal.GetSize(tFd)
		if err != nil {
			fmt.Println(err)
			return
		}

		image, _, err := image.Decode(bytes.NewReader(dat))

		if err != nil {
			fmt.Println(err)
			return
		}

		newImage := resize.Resize(uint(width), uint(height-2), image, resize.NearestNeighbor)
		rch <- newImage
	}

}
