package report_client

import "gorm.io/datatypes"

type CategoryClientResponse struct {
	ID             string
	Name           string
	Slug           string
	SearchKeywords datatypes.JSON
}
