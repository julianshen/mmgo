package state

import (
	"encoding/xml"
	"fmt"
	"math"
)

type svgFloat float64

func (v svgFloat) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: fmt.Sprintf("%.2f", round2(float64(v)))}, nil
}

func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}

type svgDoc struct {
	XMLName  xml.Name `xml:"svg"`
	XMLNS    string   `xml:"xmlns,attr"`
	ViewBox  string   `xml:"viewBox,attr"`
	Children []any    `xml:",any"`
}

type rect struct {
	XMLName xml.Name `xml:"rect"`
	X       svgFloat `xml:"x,attr"`
	Y       svgFloat `xml:"y,attr"`
	Width   svgFloat `xml:"width,attr"`
	Height  svgFloat `xml:"height,attr"`
	RX      svgFloat `xml:"rx,attr,omitempty"`
	RY      svgFloat `xml:"ry,attr,omitempty"`
	Style   string   `xml:"style,attr,omitempty"`
}

type line struct {
	XMLName   xml.Name `xml:"line"`
	X1        svgFloat `xml:"x1,attr"`
	Y1        svgFloat `xml:"y1,attr"`
	X2        svgFloat `xml:"x2,attr"`
	Y2        svgFloat `xml:"y2,attr"`
	Style     string   `xml:"style,attr,omitempty"`
	MarkerEnd string   `xml:"marker-end,attr,omitempty"`
}

type text struct {
	XMLName  xml.Name `xml:"text"`
	X        svgFloat `xml:"x,attr"`
	Y        svgFloat `xml:"y,attr"`
	Anchor   string   `xml:"text-anchor,attr,omitempty"`
	Dominant string   `xml:"dominant-baseline,attr,omitempty"`
	Style    string   `xml:"style,attr,omitempty"`
	Content  string   `xml:",chardata"`
}

type circle struct {
	XMLName xml.Name `xml:"circle"`
	CX      svgFloat `xml:"cx,attr"`
	CY      svgFloat `xml:"cy,attr"`
	R       svgFloat `xml:"r,attr"`
	Style   string   `xml:"style,attr,omitempty"`
}

type polygon struct {
	XMLName xml.Name `xml:"polygon"`
	Points  string   `xml:"points,attr"`
	Style   string   `xml:"style,attr,omitempty"`
}

type defs struct {
	XMLName xml.Name `xml:"defs"`
	Markers []marker `xml:"marker,omitempty"`
}

type marker struct {
	XMLName  xml.Name `xml:"marker"`
	ID       string   `xml:"id,attr"`
	ViewBox  string   `xml:"viewBox,attr"`
	RefX     svgFloat `xml:"refX,attr"`
	RefY     svgFloat `xml:"refY,attr"`
	Width    svgFloat `xml:"markerWidth,attr"`
	Height   svgFloat `xml:"markerHeight,attr"`
	Orient   string   `xml:"orient,attr"`
	Children []any    `xml:",any"`
}
