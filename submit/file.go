package submit

type File struct {
	Id        string `json:"_id"`
	Name      string `json:"name"`
	Reference string `json:"reference"`
	Link      string `json:"link"`
}
