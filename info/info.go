package info

import (
    "net/http"
    "net/url"
    "io/ioutil"
    "strings"

    "github.com/gophergala/aeris/format"
)

const API_INFO_URL = "http://www.youtube.com/get_video_info?hl=en_US&el=detailpage&video_id="

type Info struct {
    Id          string
    Streams     []*Stream
}

type Stream struct {
    Url         string
    signature   string
    Format      *format.YoutubeFormat
}

func NewInfo(id string) *Info {
    return &Info{
        Id: id,
    }
}

func (i *Info) Fetch() error {
    res, err := http.Get(API_INFO_URL + i.Id)
    if err != nil {
        return err
    }
    defer res.Body.Close()

    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        return err
    }
    res.Body.Close()

    rawInfo, err := url.ParseQuery(string(body))
    if err != nil {
        return err
    }

    err = i.parseStreams(rawInfo.Get("url_encoded_fmt_stream_map"))
    if err != nil {
        return err
    }

    err = i.decryptSignatures()

    return err
}

func (i *Info) parseStreams(streams string) error {
    i.Streams = nil

    formats := format.YoutubeFormats()

    for _, encodedStream := range strings.Split(streams, ",") {
        streamInfo, err := url.ParseQuery(encodedStream)
        if err != nil {
            return err
        }

        itag := streamInfo.Get("itag")
        if format, ok := formats[itag]; ok {
            stream := &Stream{
                Url: streamInfo.Get("url"),
                signature: streamInfo.Get("s"),
                Format: format,
            }

            signature := streamInfo.Get("s")
            if signature != "" {
                err = stream.buildSignatureUrl(signature)
                if err != nil {
                    return err
                }
            }

            i.Streams = append(i.Streams, stream)
        }
    }
    
    return nil
}

func (s *Stream) buildSignatureUrl(sig string) error {
    u, err := url.Parse(s.Url)
    if err != nil {
        return err
    }

    q := u.Query()

    q.Set("signature", sig)

    u.RawQuery = q.Encode()

    s.Url = u.String()

    s.signature = sig

    return nil
}
