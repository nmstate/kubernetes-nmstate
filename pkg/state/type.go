package state

type rootState struct {
	Interfaces []interface{} `json:"interfaces"`
	Routes     *routesState  `json:"routes,omitempty"`
}

type routesState struct {
	Config  []interface{} `json:"config"`
	Running []interface{} `json:"running"`
}
