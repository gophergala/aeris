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
	"strconv"
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

	// TODO: should update info with new stream information to ensure we apply the decryption
	// to the correct set of stream urls (currently not possible, as parseStream nils
	// the []*Stream slice which makes the pointer we have in the download.Download func point
	// to a Stream outside our slice, and therefore doesn't get the updated signature applied)
	// err = i.parseStreams(config.Args.UrlEncodedFmtStreamMap)
	// if err != nil {
	// 	return err
	// }

	fmt.Println(string(body[:60]))

	fmt.Println(i.Streams[0].Url)

	// body contains javascript code containing decryption info, we're about to parse those
	// and use them to decrypt signatures
	decryption, err := extractDecryption(body)
	if err != nil {
		return err
	}

	for _, stream := range i.Streams {

		stream.signature = decryption.run(stream.signature)

		err = stream.buildSignatureUrl(stream.signature)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractDecryption(js []byte) (chain, error) {

	actionString, object, err := extractMethodCalls(js)
	if err != nil {
		return nil, err
	}

	fmt.Println(actionString)
	fmt.Println(object)

	methodMapping, err := extractMethodMapping(object, js)
	if err != nil {
		return nil, err
	}

	fmt.Println("extracted", len(methodMapping), "method mappings")

	actionCalls := strings.Split(actionString, ";")

	fmt.Println("expecting", len(actionCalls), "actions in decryption chain")

	decryption, err := buildDecryptionChain(methodMapping, actionCalls)
	if err != nil {
		return decryption, err
	}

	fmt.Println("chain length =", len(decryption))

	return decryption, nil
}

var methodCallInfoRegex = regexp.MustCompile(`(?i)[a-z0-9]{2}\.([a-z0-9]{2})\([a-z],([0-9]+)\)`)

type action struct {
	method		method
	param		int
}

type chain []*action

func (c *chain) run(sig string) string {

	for _, action := range *c {
		sig = action.method.handler(sig, action.param)
	}

	return sig
}


func buildDecryptionChain(methods map[string]method, jsCalls []string) (chain, error) {

	var actions chain

	for _, call := range jsCalls {

		match := methodCallInfoRegex.FindStringSubmatch(call)
		if match == nil {
			return nil, errors.New("Could not match info extraction regex against method call")
		}

		name := match[1]
		param, _ := strconv.Atoi(match[2])

		fmt.Println(name, param)

		actions = append(actions, &action{
			method: methods[name],
			param: param,
		})
	}

	return actions, nil
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

func reverseHandler(sig string, _ int) string {

	runes := []rune(sig)

	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

func swapHandler(sig string, pos int) string {
	runes := []rune(sig)

	temp := runes[0]

	runes[0] = runes[pos % len(runes)]

	runes[pos] = temp

	return string(runes)
}

func spliceHandler(sig string, pos int) string {
	runes := []rune(sig)
	runes = runes[pos:]
	return string(runes)
}
