package report_client

type ReportClientResponse struct {
	ID              string
	Title           string
	Description     string
	Severity        string
	CategorySlug    string
	CategoryName    string
	SourceType      string
	SourceURL       string
	PrimaryPhotoURL string
	Address         string
}
