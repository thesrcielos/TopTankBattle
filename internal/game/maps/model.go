package maps

type Map struct {
	Height int     `json:"height"`
	Width  int     `json:"width"`
	Layers []Layer `json:"layers"`
}

type Layer struct {
	Data []int  `json:"data"`
	Name string `json:"name"`
}
