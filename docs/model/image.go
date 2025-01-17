package docs

type ImagesListResponse struct {
	Error  bool              `json:"error" example:"false" format:"bool"`
	Images []ImageLabelValue `json:"images"`
}

type ImageLabelValue struct {
	Label string `json:"label" example:"tensorflow/tensorflow:1.5.1"`
	Value string `json:"value" example:"tensorflow/tensorflow:1.5.1"`
}

type CommitImage struct {
	ID   string `json:"id" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
	Name string `json:"name" example:"tensorflow/tensorflow:v3"`
}
