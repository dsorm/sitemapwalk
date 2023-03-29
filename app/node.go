package app

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

var Insecure bool

type Nodes []Node

type Node struct {
	Type             uint8 // NodeTypeSitemap or NodeTypeUrl
	Loc              string
	Children         Nodes
	ChildrenNotFound bool
}

func LoadAndExpandSitemap(sitemapBytes []byte) (Node, error) {
	smxml := XMLSitemapIndex{}
	err := xml.Unmarshal(sitemapBytes, &smxml)
	if err != nil {
		return Node{}, err
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

	return result, nil
}

// TODO set lower header timeout
func (n Node) DownloadLoc() ([]byte, error) {
	var t http.Transport

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
				InsecureSkipVerify: Insecure,
			},
			ResponseHeaderTimeout: 5 * time.Second,
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
