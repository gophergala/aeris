package info

import (
	"encoding/json"
	"net/http"
	"regexp"
	"io/ioutil"
	"bytes"
	"errors"
	"fmt"
)

var configRegex = regexp.MustCompile(`ytplayer\.config = (.*);ytplayer\.load`)

const WATCH_PAGE_URL = "http://www.youtube.com/watch?v="


func (i *Info) DecryptSignatures() error {

	res, err := http.Get(WATCH_PAGE_URL + i.Id)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	res.Body.Close()

	match := configRegex.FindSubmatch(buf.Bytes())
	if match == nil {
		return errors.New("Could not match yt player config in player page")
	}

	var config = struct{
		Args struct{
			UrlEncodedFmtStreamMap string `json:"url_encoded_fmt_stream_map"`
		}
		Assets struct{
			Js string
		}
	}{}

	err = json.Unmarshal(match[1], &config)
	if err != nil {
		return err
	}

	res, err = http.Get("http:" + config.Assets.Js)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	// update info with new stream information to ensure we apply the decryption
	// to the correct set of stream urls
	err = i.parseStreams(config.Args.UrlEncodedFmtStreamMap)
	if err != nil {
		return err
	}

	fmt.Println(string(body[:60]))

	fmt.Println(i.Streams[0].Url)

	// body contains javascript code containing decryption info, we're about to parse those
	// and use them to decrypt signatures

	return nil
}
