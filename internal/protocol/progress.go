package protocol

// Progress tracks work item counts in a session.
type Progress struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Assigned int `json:"assigned"`
	Done     int `json:"done"`
	Failed   int `json:"failed"`
	Blocked  int `json:"blocked"`
	Canceled int `json:"canceled"`
}
