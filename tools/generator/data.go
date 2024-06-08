package main

type Data struct {
	PackageName  string
	FileName     string
	ApiVersion   string
	SpecFields   []Field
	StatusFields []Field
}

type Field struct {
	Name     string
	Type     string
	JsonName string
}

func parseSchema(file string) Data {
	// TODO: implement schema parsing
	return Data{
		PackageName: "prc_com_bucket_v1",
		FileName:    "crd.go",
		ApiVersion:  "prc.com/v1",
		SpecFields: []Field{
			{
				Name:     "Prefix",
				Type:     "string",
				JsonName: "prefix",
			},
			{
				Name:     "Tags",
				Type:     "map[string]string",
				JsonName: "tags",
			},
		},
		StatusFields: []Field{
			{
				Name:     "Arn",
				Type:     "*string",
				JsonName: "arn",
			},
		},
	}
}
