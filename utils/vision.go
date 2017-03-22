package utils

// Size defines the size of a rectangle
type Size struct {
	W int `json:"w"`
	H int `json:"h"`
}

// Square calculates the size of area
func (s *Size) Square() int {
	return s.W * s.H
}

// Pos defines the position of a point
type Pos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Rect defines a rectangle area
type Rect struct {
	Pos
	Size
}

// Object is detected object
type Object struct {
	Type  string   `json:"type"`
	Range Rect     `json:"range"`
	Rate  *float32 `json:"rate"`
}

// Result is result of vision analytics
type Result struct {
	Size    Size      `json:"size"`
	Objects []*Object `json:"objects"`
}

// ByRate is sort algo by rate
type ByRate []*Object

// Len implements sort.Interface
func (b ByRate) Len() int {
	return len(b)
}

// Swap implements sort.Interface
func (b ByRate) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// Less implements sort.Interface
func (b ByRate) Less(i, j int) bool {
	if b[i].Rate == nil {
		return b[j].Rate != nil
	}
	if b[j].Rate == nil {
		return false
	}
	return *b[i].Rate < *b[j].Rate
}
