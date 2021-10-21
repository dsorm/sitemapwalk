package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	NodeTypeUndetermined = iota
	NodeTypeRoot
	NodeTypeSitemap
	NodeTypeUrl
	NodeTypeError
)

var Debug = false

type Nodes []Node

type Node struct {
	Type             uint8 // NodeTypeSitemap or NodeTypeUrl
	Loc              string
	Children         Nodes
	ChildrenNotFound bool
}

type XMLSitemapIndex struct {
	XMLName  xml.Name     `xml:"sitemapindex"`
	Sitemaps []XMLSitemap `xml:"sitemap"`
}

type XMLSitemap struct {
	XMLName xml.Name `xml:"sitemap"`
	Loc     string   `xml:"loc"`
}

type XMLUrlSet struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []XMLUrl `xml:"url"`
}

type XMLUrl struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
}

func (x XMLSitemapIndex) ParseAndAttachToNode(n *Node) {
	if len(x.Sitemaps) == 0 {
		return
	}

	children := make(Nodes, len(x.Sitemaps), len(x.Sitemaps))

	for k, v := range x.Sitemaps {
		children[k] = Node{
			Type:             NodeTypeSitemap,
			Loc:              v.Loc,
			Children:         nil,
			ChildrenNotFound: false,
		}
	}

	n.Children = children
}

func (x XMLUrlSet) ParseAndAttachToNode(n *Node) {
	// TODO
	if len(x.Urls) == 0 {
		return
	}

	children := make(Nodes, len(x.Urls), len(x.Urls))

	for k, v := range x.Urls {
		children[k] = Node{
			Type:     NodeTypeUrl,
			Loc:      v.Loc,
			Children: nil,
			// Urls can't have children, so it's sure that no children will be found
			ChildrenNotFound: true,
		}
	}

	n.Children = children
}

// TODO set lower header timeout
func (n Node) DownloadLoc() ([]byte, error) {
	var t http.Transport

	// allow https without valid certificates for debugging
	if Debug {
		t = http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Proxy: func(*http.Request) (*url.URL, error) {
				return &url.URL{
					Scheme: "https://",
					Host:   "localhost:8445",
				}, nil
			},
			ResponseHeaderTimeout: 2 * time.Second,
		}
	} else {
		t = http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
			ResponseHeaderTimeout: 2 * time.Second,
		}
	}
	client := http.Client{
		Transport:     &t,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       10 * time.Second,
	}

	req, err := http.NewRequest("GET", n.Loc, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "sitemapwalk/dev")

	resp, err := client.Get(n.Loc)
	if err != nil {
		return nil, err
	}

	//magicNumbers := make([]byte, 2, 2)
	//bytesRead, err ;= resp.Body.Read(magicNumbers)
	//if n != 2 {
	//	return nil, errors.New("Error while reading magic numbers (first two numbers) from the page: not enough data")
	//}

	locBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile("respbodyraw.txt", locBytes, 0644)
	if err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if IsGzip(locBytes) {
		locReader := bytes.NewReader(locBytes)

		gzipReader, err := gzip.NewReader(locReader)
		if err != nil {
			return nil, err
		}

		locBytes, err = io.ReadAll(gzipReader)
		if err != nil {
			return nil, err
		}
	}

	fmt.Println(time.Now().String(), "; GET", n.Loc, ", successful")
	//time.Sleep(25*time.Millisecond)
	return locBytes, nil
}

func IsGzip(b []byte) bool {
	if len(b) < 2 {
		return false
	}

	return b[0] == 0x1f && b[1] == 0x8b
}

func ContainsSitemaps(b []byte) bool {
	return strings.Contains(string(b), "<sitemap>")
}

func ParseSitemap(b []byte) (XMLSitemapIndex, error) {
	x := XMLSitemapIndex{}
	err := xml.Unmarshal(b, &x)
	if err != nil {
		return XMLSitemapIndex{}, err
	}
	return x, nil
}

func ContainsUrls(b []byte) bool {
	return strings.Contains(string(b), "<url>")
}

func ParseUrls(b []byte) (XMLUrlSet, error) {
	x := XMLUrlSet{}
	err := xml.Unmarshal(b, &x)
	if err != nil {
		return XMLUrlSet{}, err
	}
	return x, nil
}

// Expand expands the node tree by one level through acquiring additional information
// Returns the modified node
func (n Node) Expand() Node {
	if n.Type == NodeTypeUrl || n.ChildrenNotFound {
		return n
	}

	if n.Children == nil {
		locBody, err := n.DownloadLoc()
		if err != nil {
			fmt.Printf("Error happened while expanding this node:\n%v\nerror:\n%v\n", n, err)
			n.Type = NodeTypeError
			return n
		}

		if ContainsUrls(locBody) {
			xus, err := ParseUrls(locBody)
			if err != nil {
				fmt.Printf("Error happened while expanding this node:\n%v\nerror:\n%v\n", n, err)
				n.Type = NodeTypeError
				return n
			}
			xus.ParseAndAttachToNode(&n)
			return n
		}

		if ContainsSitemaps(locBody) {
			xsi, err := ParseSitemap(locBody)
			if err != nil {
				fmt.Printf("Error happened while expanding this node:\n%v\nerror:\n%v\n", n, err)
				n.Type = NodeTypeError
				return n
			}
			xsi.ParseAndAttachToNode(&n)
		}

	}

	for k, v := range n.Children {
	preswitch:
		switch v.Type {
		case NodeTypeUndetermined:
			childLocBody, err := v.DownloadLoc()
			if err != nil {
				fmt.Printf("Error happened while expanding this node:\n%v\nerror:\n%v\n", n, err)
				n.Type = NodeTypeError
				return n
			}

			if ContainsUrls(childLocBody) {
				n.Children[k].Type = NodeTypeUrl
			}
			if ContainsSitemaps(childLocBody) {
				n.Children[k].Type = NodeTypeSitemap
			}
			goto preswitch
		case NodeTypeUrl:
			continue
		case NodeTypeSitemap:
			n.Children[k] = v.Expand()
		}

	}
	return n
}

func (n Node) CallForEachUrl(f func(urlNode Node)) {
	if n.Children == nil {
		return
	}

	for _, v := range n.Children {
		if v.Type == NodeTypeUrl {
			f(v)
		}
	}
}

func (n Node) SendForEachUrl(ch chan<- Node) {
	if n.Children == nil {
		return
	}

	for _, v := range n.Children {
		if v.Type == NodeTypeUrl {
			ch <- v
		}

		if v.Type == NodeTypeSitemap {
			v.SendForEachUrl(ch)
		}
	}
}

func loadAndExpandSitemap() Node {
	smBytes, err := os.ReadFile("sitemaps.xml")
	if err != nil {
		panic(err)
	}
	smxml := XMLSitemapIndex{}
	err = xml.Unmarshal(smBytes, &smxml)
	if err != nil {
		panic(err)
	}
	rootNode := Node{
		Type:             NodeTypeRoot,
		Loc:              "",
		Children:         nil,
		ChildrenNotFound: false,
	}
	smxml.ParseAndAttachToNode(&rootNode)

	// dark magic of recursion
	result := rootNode.Expand()

	fmt.Println("writing result as json")
	jsonBytes, _ := json.MarshalIndent(result, "", "	")
	err = os.WriteFile(time.Now().Format("2006-01-02_15-04-05 ")+"result.json", jsonBytes, 0644)

	fmt.Println("done")
	return result
}
func main() {
	Debug = false
	appCtx, appCtxCancel := context.WithCancel(context.Background())
	defer appCtxCancel()
	var rootNode Node
	rootNode = loadAndExpandSitemap()
	// fileBytes, err := os.ReadFile("2021-10-18_22-40-16 result.json")
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = json.Unmarshal(fileBytes, &rootNode)
	//if err != nil {
	//	panic(err)
	//}

	ch := make(chan Node, 64)

	sendCtx, sendCtxCancel := context.WithCancel(appCtx)

	go func() {
		rootNode.SendForEachUrl(ch)
		sendCtxCancel()
		fmt.Println("context cancelled")
	}()

	fmt.Println()
	i := uint64(0)
	for {
		select {
		case node := <-ch:
			i++
			if (i % 100) == 0 {
				fmt.Println(node)
			}
			fmt.Printf("\r%v", i)

		case <-sendCtx.Done():
			fmt.Println("accepted context cancel, quitting...")
			fmt.Println("items counted:", i)
			return
		}

	}

}
