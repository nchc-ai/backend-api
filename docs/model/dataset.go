package docs

type DatasetsListResponse struct {
	Error    bool                `json:"error"`
	Datasets []DatasetLabelValue `json:"datasets"`
}

type DatasetLabelValue struct {
	Label string `json:"label" example:"cifar-10"`
	Value string `json:"value" example:"dataset-cifar-10"`
}
