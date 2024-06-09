package main

type Data struct {
	PackageName string
	// CRD fileds
	CrdApiVersion   string
	CrdSpecFields   []CrdField
	CrdStatusFields []CrdField
	// Terraform Resource Model fields
	RmTypeName string
	RmFields   []rmField
}

type CrdField struct {
	Name     string
	Type     string
	JsonName string
}

type rmField struct {
	Name      string
	Type      string
	TfsdkName string
}

func parseSchema(file string) (Data, error) {
	// TODO: implement schema parsing
	return Data{
		PackageName:   "prc_com_bucket_v1",
		CrdApiVersion: "prc.com/v1",
		CrdSpecFields: []CrdField{
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
		CrdStatusFields: []CrdField{
			{
				Name:     "Arn",
				Type:     "*string",
				JsonName: "arn",
			},
		},
		RmTypeName: "bucketResourceModel",
		RmFields: []rmField{
			{
				Name:      "Prefix",
				Type:      "types.String",
				TfsdkName: "prefix",
			},
			{
				Name:      "Tags",
				Type:      "types.Map",
				TfsdkName: "tags",
			},
			{
				Name:      "Arn",
				Type:      "types.String",
				TfsdkName: "arn",
			},
		},
	}, nil
}
