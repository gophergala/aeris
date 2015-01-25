package main

import (
	"flag"
	"os"
	"fmt"
	"strings"
	"io"

	"github.com/gophergala/aeris/download"
	"github.com/gophergala/aeris/info"
)

var (
	inputItag int
)

func main() {

	flag.IntVar(&inputItag, "i:i", -1, "itag of the stream to download")
	flag.Parse()

	if flag.NArg() == 0 {
		printUsage()
		os.Exit(0)
	}

	cmd := flag.Arg(0)

	switch cmd {

		// 0: get
		case "get":

			if flag.NArg() < 2 {
				printUsageCommand("get")
				os.Exit(1)
			}

			id := strings.TrimLeft(flag.Arg(0), "\\")

			i, err := fetchInfo(id)
			if err != nil {
				fmt.Println("there was an error fetching info for the video", id)
				os.Exit(1)
			}

			var downloadStream *info.Stream

			if inputItag != -1 {

				// see if we can satisfy the user
				for _, stream := range i.Streams() {
					if stream.Format.Itag == inputItag {
						downloadStream = stream
						break
					}
				}

				fmt.Println("this video doesn't have itag", inputItag, ". picking default stream (the best one available).")
			}

			// pick the best stream by default
			if downloadStream == nil {
				downloadStream = i.Streams()[0]
			}

			var output io.Writer
			if flag.NArg() > 1 {

				if flag.Arg(1) == "-" {

					output = os.Stdout

				} else {

					output, err := os.Create(flag.Arg(1))
					if err != nil {
						fmt.Println("failed to create file")
						os.Exit(1)
					}
					defer output.Close()
				}

			} else {

				ext, err := downloadStream.Format.Extension()
				if err != nil {
					// no extension, I don't have all possibilities listed in (*Format).Extension()
					ext = ""
				}

				output, err := os.Create(i.Id + ext)
				if err != nil {
					fmt.Println("failed to create file")
					os.Exit(1)
				}
				defer output.Close()

			}

			download.Download(i, downloadStream, output)

		case "info":
			showVideoInfo()

		case "help":

			if flag.NArg() == 2 {
				printUsageCommand(flag.Arg(1))
			} else {
				printUsage()
			}

		default:
			printUsage()

	}

}

func downloadVideo(id string) error {

	videoInfo, err := fetchInfo(id)
	if err != nil {
		return err
	}

	stream := videoInfo.Streams()[0]
	extension, err := stream.Format.Extension()
	if err != nil {
		return err
	}

	fd, err := os.Create(videoInfo.Id + extension)
	if err != nil {
		return err
	}
	defer fd.Close()

	err = download.Download(videoInfo, stream, fd)
	if err != nil {
		return err
	}

	return nil
}

func fetchInfo(id string) (*info.Info, error) {

	videoInfo := info.NewInfo(id)

	err := videoInfo.Fetch()
	if err != nil {
		return nil, err
	}

	return videoInfo, err
}

func showVideoInfo() {

}

func printHeader() {
	fmt.Println("")
	fmt.Println("    aeris - (c) Gijsbrecht Hermans 2015")
	fmt.Println("")
}

func printSubHeader() {
	fmt.Println("    Easily download videos in various formats from YouTube")
	fmt.Println("")
}

func printUsage() {

	printHeader()

	printSubHeader()

	fmt.Println("Usage:")
	fmt.Println("    aeris <command> [arguments...]")
	fmt.Println("")
	fmt.Println("A List of possible commands:")
	fmt.Println("")
	fmt.Println("    get         download a video from YouTube")
	fmt.Println("    info        fetch technical info about a video from YouTube")
	fmt.Println("    help        help with aeris")
	fmt.Println("")
	fmt.Println("To view information about the usage of these commands execute")
	fmt.Println("")
	fmt.Println("    aeris help <command>")
}

func printUsageCommand(cmd string) {

	printHeader()

	printSubHeader()

	fmt.Println("Usage:")

	switch cmd {
		case "get":
			fmt.Println("    get <video-id>")
			fmt.Println("")
			fmt.Println("Available flags:")
			fmt.Println("    -i:i        itag identifying a YouTube stream to download")

		case "info":

		case "help":
			fmt.Println("")
			fmt.Println("    woah! help-ception!")
	}

}
