package app

//JsonData export to clent
type JsonData struct {
	Code   int         `json:"code"`
	Title  interface{} `json:"title"`
	Attach interface{} `json:"attach,omitempty"`
	Data   interface{} `json:"data"`
	Other  interface{} `json:"other,omitempty"`
}

//ExportData ExportData
func ExportData(code int, title interface{}, data ...interface{}) *JsonData {
	var resultData JsonData
	resultData.Code = code
	resultData.Title = title
	resultData.Data = data[0]

	if len(data) > 1 {
		resultData.Attach = data[1]
		if len(data) > 2 {
			resultData.Other = data[2]
		}

	}
	// if EnvName == EnvProd && code == 500 {
	// 	resultData.Title = "Error Information"
	// 	resultData.Data = ""
	// }
	return &resultData
}
