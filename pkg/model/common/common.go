package common

type LabelValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type LabelIntValue struct {
	Label string `json:"label"`
	Value *int32 `json:"value"`
}
