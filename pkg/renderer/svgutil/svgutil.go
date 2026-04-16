// Package svgutil provides shared SVG XML types and helpers used
// across all diagram renderers.
package svgutil

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
)

type Float float64

func (v Float) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: fmt.Sprintf("%.2f", Round2(float64(v)))}, nil
}

func Round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}

func Sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func MarshalDoc(doc any) ([]byte, error) {
	svgBytes, err := xml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	xmlDecl := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	return append(xmlDecl, svgBytes...), nil
}

type Doc struct {
	XMLName  xml.Name `xml:"svg"`
	XMLNS    string   `xml:"xmlns,attr"`
	ViewBox  string   `xml:"viewBox,attr"`
	Children []any    `xml:",any"`
}

type Rect struct {
	XMLName xml.Name `xml:"rect"`
	X       Float    `xml:"x,attr"`
	Y       Float    `xml:"y,attr"`
	Width   Float    `xml:"width,attr"`
	Height  Float    `xml:"height,attr"`
	RX      Float    `xml:"rx,attr,omitempty"`
	RY      Float    `xml:"ry,attr,omitempty"`
	Style   string   `xml:"style,attr,omitempty"`
}

type Line struct {
	XMLName   xml.Name `xml:"line"`
	X1        Float    `xml:"x1,attr"`
	Y1        Float    `xml:"y1,attr"`
	X2        Float    `xml:"x2,attr"`
	Y2        Float    `xml:"y2,attr"`
	Style     string   `xml:"style,attr,omitempty"`
	MarkerEnd string   `xml:"marker-end,attr,omitempty"`
}

type Text struct {
	XMLName  xml.Name `xml:"text"`
	X        Float    `xml:"x,attr"`
	Y        Float    `xml:"y,attr"`
	Anchor   string   `xml:"text-anchor,attr,omitempty"`
	Dominant string   `xml:"dominant-baseline,attr,omitempty"`
	Style    string   `xml:"style,attr,omitempty"`
	Content  string   `xml:",chardata"`
}

type Circle struct {
	XMLName xml.Name `xml:"circle"`
	CX      Float    `xml:"cx,attr"`
	CY      Float    `xml:"cy,attr"`
	R       Float    `xml:"r,attr"`
	Style   string   `xml:"style,attr,omitempty"`
}

type Polygon struct {
	XMLName xml.Name `xml:"polygon"`
	Points  string   `xml:"points,attr"`
	Style   string   `xml:"style,attr,omitempty"`
}

type Polyline struct {
	XMLName xml.Name `xml:"polyline"`
	Points  string   `xml:"points,attr"`
	Style   string   `xml:"style,attr,omitempty"`
}

type Path struct {
	XMLName   xml.Name `xml:"path"`
	D         string   `xml:"d,attr"`
	Style     string   `xml:"style,attr,omitempty"`
	MarkerEnd string   `xml:"marker-end,attr,omitempty"`
}

type Defs struct {
	XMLName xml.Name `xml:"defs"`
	Markers []Marker `xml:"marker,omitempty"`
}

type Marker struct {
	XMLName  xml.Name `xml:"marker"`
	ID       string   `xml:"id,attr"`
	ViewBox  string   `xml:"viewBox,attr"`
	RefX     Float    `xml:"refX,attr"`
	RefY     Float    `xml:"refY,attr"`
	Width    Float    `xml:"markerWidth,attr"`
	Height   Float    `xml:"markerHeight,attr"`
	Orient   string   `xml:"orient,attr"`
	Children []any    `xml:",any"`
}

func ViewBox(w, h float64) string {
	return fmt.Sprintf("0 0 %.2f %.2f", w, h)
}

func XMLDecl() []byte {
	return []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
}

func MarshalSVG(doc Doc) ([]byte, error) {
	raw, err := xml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return bytes.Join([][]byte{XMLDecl(), raw}, nil), nil
}
