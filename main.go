package main

import (
	"flag"
	"os"
	"strings"

	"github.com/gophergala/aeris/download"
	"github.com/gophergala/aeris/info"
)

func fetchInfo(id string) (*info.Info, error) {

	videoInfo := info.NewInfo(id)

	err := videoInfo.Fetch()
	if err != nil {
		return nil, err
	}

	return videoInfo, err
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

func main() {

	flag.Parse()

	id := strings.TrimLeft(flag.Arg(0), "\\")

	downloadVideo(id)

}
