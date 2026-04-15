package flowchart

import (
	"encoding/xml"
	"fmt"
	"math"
)

// svgFloat is a float64 that marshals to XML rounded to 2 decimal
// places. This eliminates cross-platform differences in float
// formatting (macOS emits "41.099999999999994" while Linux emits
// "41.1") so golden-file tests are byte-identical everywhere.
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

type SVG struct {
	XMLName  xml.Name `xml:"svg"`
	XMLNS    string   `xml:"xmlns,attr"`
	ViewBox  string   `xml:"viewBox,attr"`
	Width    string   `xml:"width,attr,omitempty"`
	Height   string   `xml:"height,attr,omitempty"`
	Children []any    `xml:",any"`
}

type StyleEl struct {
	XMLName xml.Name `xml:"style"`
	Content string   `xml:",chardata"`
}

type Group struct {
	XMLName   xml.Name `xml:"g"`
	ID        string   `xml:"id,attr,omitempty"`
	Class     string   `xml:"class,attr,omitempty"`
	Style     string   `xml:"style,attr,omitempty"`
	Transform string   `xml:"transform,attr,omitempty"`
	Children  []any    `xml:",any"`
}

type Rect struct {
	XMLName xml.Name `xml:"rect"`
	X       svgFloat `xml:"x,attr"`
	Y       svgFloat `xml:"y,attr"`
	Width   svgFloat `xml:"width,attr"`
	Height  svgFloat `xml:"height,attr"`
	RX      svgFloat `xml:"rx,attr,omitempty"`
	RY      svgFloat `xml:"ry,attr,omitempty"`
	Style   string   `xml:"style,attr,omitempty"`
	Class   string   `xml:"class,attr,omitempty"`
}

type Circle struct {
	XMLName xml.Name `xml:"circle"`
	CX      svgFloat `xml:"cx,attr"`
	CY      svgFloat `xml:"cy,attr"`
	R       svgFloat `xml:"r,attr"`
	Style   string   `xml:"style,attr,omitempty"`
	Class   string   `xml:"class,attr,omitempty"`
}

type Polygon struct {
	XMLName xml.Name `xml:"polygon"`
	Points  string   `xml:"points,attr"`
	Style   string   `xml:"style,attr,omitempty"`
	Class   string   `xml:"class,attr,omitempty"`
}

type Polyline struct {
	XMLName xml.Name `xml:"polyline"`
	Points  string   `xml:"points,attr"`
	Style   string   `xml:"style,attr,omitempty"`
	Class   string   `xml:"class,attr,omitempty"`
}

type Path struct {
	XMLName     xml.Name `xml:"path"`
	D           string   `xml:"d,attr"`
	Style       string   `xml:"style,attr,omitempty"`
	Class       string   `xml:"class,attr,omitempty"`
	Fill        string   `xml:"fill,attr,omitempty"`
	MarkerEnd   string   `xml:"marker-end,attr,omitempty"`
	MarkerStart string   `xml:"marker-start,attr,omitempty"`
}

type Line struct {
	XMLName     xml.Name `xml:"line"`
	X1          svgFloat `xml:"x1,attr"`
	Y1          svgFloat `xml:"y1,attr"`
	X2          svgFloat `xml:"x2,attr"`
	Y2          svgFloat `xml:"y2,attr"`
	Style       string   `xml:"style,attr,omitempty"`
	Class       string   `xml:"class,attr,omitempty"`
	MarkerEnd   string   `xml:"marker-end,attr,omitempty"`
	MarkerStart string   `xml:"marker-start,attr,omitempty"`
}

type Text struct {
	XMLName    xml.Name `xml:"text"`
	X          svgFloat `xml:"x,attr"`
	Y          svgFloat `xml:"y,attr"`
	Anchor     string   `xml:"text-anchor,attr,omitempty"`
	Dominant   string   `xml:"dominant-baseline,attr,omitempty"`
	FontSize   svgFloat `xml:"font-size,attr,omitempty"`
	FontFamily string   `xml:"font-family,attr,omitempty"`
	Style      string   `xml:"style,attr,omitempty"`
	Class      string   `xml:"class,attr,omitempty"`
	Content    string   `xml:",chardata"`
}

type Defs struct {
	XMLName xml.Name `xml:"defs"`
	Markers []Marker `xml:"marker,omitempty"`
}

type Marker struct {
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
