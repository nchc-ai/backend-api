package docs

type HealthKubernetesResponse struct {
	Error   bool   `json:"error" example:"false" format:"bool"`
	Message []Node `json:"message"`
}

type Node struct {
	Name   string `json:"name" example:"10.0.1.85"`
	Status string `json:"status" example:"Ready"`
}


type GenericDBRequest struct {
	Message string `json:"message" example:"test"`
}

type GenericDBResponse struct {
	Error   bool   `json:"error" example:"false" format:"bool"`
	Message string `json:"message" example:"test" format:"string"`
}

type HealthDatabaseResponse struct {
	GenericDBResponse
	Tables []string `json:"tables" example:"jobs,courses"`
}
