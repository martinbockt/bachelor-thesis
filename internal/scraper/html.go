package scraper

import (
	"strings"

	"golang.org/x/net/html"
)

func findBodyNode(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findBodyNode(c); result != nil {
			return result
		}
	}

	return nil
}

func removeComments(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.CommentNode {
			n.RemoveChild(c)
		} else {
			removeComments(c)
		}
		c = next
	}
}

func normalizeWhitespace(n *html.Node) {
	if n.Type == html.TextNode {
		n.Data = strings.Join(strings.Fields(n.Data), " ")
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		normalizeWhitespace(c)
	}
}

func removeUnwantedTags(n *html.Node, tagName string) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		removeUnwantedTags(c, tagName)
		if c.Type == html.ElementNode && c.Data == tagName {
			n.RemoveChild(c)
		}
		c = next
	}
}

// removeAllAttributesExceptImportant removes all attributes except for important ones
func removeAllAttributesExceptImportant(n *html.Node) {
	if n.Type == html.ElementNode {
		var newAttrs []html.Attribute
		for _, attr := range n.Attr {
			if isImportantAttribute(attr) {
				newAttrs = append(newAttrs, attr)
			} else if attr.Key == "style" {
				// keep background images in style attribute
				newStyle := filterStyleAttribute(attr.Val)
				if newStyle != "" {
					attr.Val = newStyle
					newAttrs = append(newAttrs, attr)
				}
			}
		}
		n.Attr = newAttrs
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		removeAllAttributesExceptImportant(c)
	}
}

// isImportantAttribute checks if an attribute is important and should be kept.
func isImportantAttribute(attr html.Attribute) bool {
	switch attr.Key {
	case "id", "name", "href", "src", "alt", "title", "type", "value", "srcset":
		return true
	default:
		// Keep data-* attributes
		if strings.HasPrefix(attr.Key, "data-") {
			return true
		}

		return false
	}
}

func filterStyleAttribute(styleContent string) string {
	var filteredDeclarations []string
	declarations := strings.Split(styleContent, ";")
	for _, decl := range declarations {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		// Check if the declaration contains "background-image"
		if strings.Contains(decl, "background-image") {
			filteredDeclarations = append(filteredDeclarations, decl)
		}
	}

	return strings.Join(filteredDeclarations, "; ")
}

// removeEmptyElements recursively removes all elements that have no content (including text, attributes, or child nodes).
func removeEmptyElements(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		removeEmptyElements(c) // Recursively remove empty elements in children
		if isEmptyElement(c) {
			// Remove the current node
			n.RemoveChild(c)
		}
		c = next
	}
}

// isEmptyElement checks if a node is empty (has no children or text content).
func isEmptyElement(n *html.Node) bool {
	if n.Type == html.ElementNode {
		// <img> tags are considered empty if they have no src attribute
		if n.Data == "img" {
			for _, attr := range n.Attr {
				if (attr.Key == "src" || attr.Key == "srcset") && strings.TrimSpace(attr.Val) != "" {
					return false // The <img> tag is not empty if it has a valid src attribute
				}
			}

			return true // The <img> tag is empty if it has no src attribute
		}
		// Element is empty if it has no children and no significant text
		if n.FirstChild == nil {
			return true
		}
		// Check if all children are empty text nodes or empty elements
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.TextNode || strings.TrimSpace(c.Data) != "" {
				return false
			}
		}

		return true
	}

	return false
}

// removeEmptyLinks removes <a> tags with an empty href or href="#".
func removeEmptyLinks(n *html.Node) {
	var prev *html.Node
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.ElementNode && c.Data == "a" {
			for _, attr := range c.Attr {
				if attr.Key == "href" && (attr.Val == "" || attr.Val == "#") {
					if prev != nil {
						prev.NextSibling = next
					} else {
						n.FirstChild = next
					}
					c.Parent = nil

					break
				}
			}
		} else {
			removeEmptyLinks(c)
			prev = c
		}
		c = next
	}
}
