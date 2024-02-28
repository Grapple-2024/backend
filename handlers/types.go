package handlers

type GetResponse struct {
	Data             any     `json:"data"`
	LastEvaluatedKey *string `json:"lastEvaluatedKey"`
	Count            int32   `json:"count"`
}
