package flowchart

import "encoding/xml"

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
	X       float64  `xml:"x,attr"`
	Y       float64  `xml:"y,attr"`
	Width   float64  `xml:"width,attr"`
	Height  float64  `xml:"height,attr"`
	RX      float64  `xml:"rx,attr,omitempty"`
	RY      float64  `xml:"ry,attr,omitempty"`
	Style   string   `xml:"style,attr,omitempty"`
	Class   string   `xml:"class,attr,omitempty"`
}

type Circle struct {
	XMLName xml.Name `xml:"circle"`
	CX      float64  `xml:"cx,attr"`
	CY      float64  `xml:"cy,attr"`
	R       float64  `xml:"r,attr"`
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
	X1          float64  `xml:"x1,attr"`
	Y1          float64  `xml:"y1,attr"`
	X2          float64  `xml:"x2,attr"`
	Y2          float64  `xml:"y2,attr"`
	Style       string   `xml:"style,attr,omitempty"`
	Class       string   `xml:"class,attr,omitempty"`
	MarkerEnd   string   `xml:"marker-end,attr,omitempty"`
	MarkerStart string   `xml:"marker-start,attr,omitempty"`
}

type Text struct {
	XMLName    xml.Name `xml:"text"`
	X          float64  `xml:"x,attr"`
	Y          float64  `xml:"y,attr"`
	Anchor     string   `xml:"text-anchor,attr,omitempty"`
	Dominant   string   `xml:"dominant-baseline,attr,omitempty"`
	FontSize   float64  `xml:"font-size,attr,omitempty"`
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
	RefX     float64  `xml:"refX,attr"`
	RefY     float64  `xml:"refY,attr"`
	Width    float64  `xml:"markerWidth,attr"`
	Height   float64  `xml:"markerHeight,attr"`
	Orient   string   `xml:"orient,attr"`
	Children []any    `xml:",any"`
}
