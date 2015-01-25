package info

import (
	"encoding/json"
	"net/http"
	"regexp"
	"io/ioutil"
	"bytes"
	"errors"
	"fmt"
	"strings"
)

var configRegex = regexp.MustCompile(`ytplayer\.config = (.*);ytplayer\.load`)

var actionsExtractRegex = regexp.MustCompile(`(?i)[a-z]=[a-z]\.split\(""\);((?:([a-z]{2})\.[a-z0-9]{2}\([a-z],[0-9]+\);)+)return [a-z]\.join\(""\)`)


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

	fmt.Println("http:" + config.Assets.Js)

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
	err = extractDecryption(body)
	if err != nil {
		return err
	}

	return nil
}

func extractDecryption(js []byte) error {

	actionString, object, err := extractMethodCalls(js)
	if err != nil {
		return err
	}

	fmt.Println(actionString)
	fmt.Println(object)

	methodMapping, err := extractMethodMapping(object, js)
	if err != nil {
		return err
	}

	fmt.Println("extracted", len(methodMapping), "method mappings")

	for name, method := range methodMapping {
		fmt.Println(method.name, name, method.definition, method.handler("", -1))
	}

	return nil
}

func objectMethodExtractRegex(objectName string) (*regexp.Regexp, error) {

	var methodArray []string
	for _, regexStr := range methodsRegex {
		methodArray = append(methodArray, regexStr)
	}

	regex, err := regexp.Compile(`(?i)var ` + objectName + `=\{((?:(?:` + strings.Join(methodArray, "|") + `)(?:,|\})?)+)`)

	return regex, err
}

type method struct {
	name		string
	definition	string
	handler		handler
}

func extractMethodMapping(object string, js []byte) (map[string]method, error) {

	regex, err := objectMethodExtractRegex(object)
	if err != nil {
		return nil, err
	}

	match := regex.FindSubmatch(js)
	if match == nil {
		return nil, errors.New("Couldn't match object method extraction regex against js body")
	}

	definitions := string(match[1])

	fmt.Println(definitions)

	methods := make(map[string]method)

	for regex, handler := range methodsRegexToHandler {
		match := regex.FindStringSubmatchIndex(definitions)
		if match != nil {

			definition := definitions[match[0]:match[1]]

			name := definitions[match[0]:match[0]+2]

			fmt.Println("def " + name + " = " + definition)

			methods[name] = method{
				name: name,
				definition: definition,
				handler: handler,
			}
		}
	}

	return methods, nil
}

// run with i modifier!
var methodsRegex = map[string]string {
	"reverse": `([a-z0-9]{2}):function\([a-z]\)\{[a-z]\.reverse\(\)\}`,
	"swap": `([a-z0-9]{2}):function\([a-z],[a-z]\)\{var [a-z]=[a-z]\[[0-9]\];[a-z]\[[0-9]\]=[a-z]\[[a-z]%[a-z]\.length\];[a-z]\[[a-z]\]=[a-z]\}`,
	"splice": `([a-z0-9]{2}):function\([a-z],[a-z]\)\{[a-z]\.splice\([0-9],[a-z]\)\}`,
}

type handler func(in string, param int) string

var methodsRegexToHandler = map[*regexp.Regexp]handler {
	regexp.MustCompile(`(?i)` + methodsRegex["reverse"]): reverseHandler,
	regexp.MustCompile(`(?i)` + methodsRegex["swap"]): swapHandler,
	regexp.MustCompile(`(?i)` + methodsRegex["splice"]): spliceHandler,
}

func extractMethodCalls(js []byte) (string, string, error) {
	match := actionsExtractRegex.FindSubmatch(js)
	if match == nil {
		return "", "", errors.New("Could not match action extraction regex against js body")
	}

	actionString := string(match[1])
	object := string(match[2])

	actionString = strings.Trim(actionString, ";")

	return actionString, object, nil
}

func reverseHandler(in string, param int) string {
	return "REVERSE"
}

func swapHandler(in string, param int) string {
	return "SWAP"
}

func spliceHandler(in string, param int) string {
	return "SPLICE"
}