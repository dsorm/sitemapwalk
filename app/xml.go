package app

import (
	"encoding/xml"
	"strings"
)

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
