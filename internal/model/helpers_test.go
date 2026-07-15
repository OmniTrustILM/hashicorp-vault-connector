package model

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestUnmarshalAttributeValue(t *testing.T) {
	result := UnmarshalAttributesValues([]byte(JSON_STRING_VALUE_ARR))
	content := GetAttributeFromArrayByUUID(AUTHORITY_URL_ATTR, result).GetContent()[0]
	URL := content.GetData().(string)
	fmt.Println(URL)
}

func TestUnmarshalAttribute(t *testing.T) {
	result := UnmarshalAttributes([]byte(JSON_STRING_ARR))
	fmt.Println(result)
	resultString, _ := json.Marshal(result)
	var unmarshaled []any
	err := json.Unmarshal([]byte(JSON_STRING_ARR), &unmarshaled)
	if err != nil {
		return
	}
	marshaled, _ := json.Marshal(unmarshaled)
	if equal, err := compareJSON(string(marshaled), string(resultString)); err != nil {
		t.Fatalf("Error comparing JSON:")
	} else if equal {
		fmt.Println("JSON strings are equal")
	} else {
		t.Fatalf("JSON strings are not equal")
	}

}

func compareJSON(s1, s2 string) (bool, error) {
	var o1 any
	var o2 any

	err := json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, err
	}

	return reflect.DeepEqual(o1, o2), nil
}

const (
	JSON_STRING_ARR = `
[
{
"name": "authority_url",
"description": "string",
"type": "data",
"content": [
{
"reference": "string",
"data": "bola"
}
],
"contentType": "string",
"properties": {
"label": "Attribute Name",
"visible": true,
"group": "requiredAttributes",
"required": false,
"readOnly": false,
"list": false,
"multiSelect": false,
"extensibleList": false
}
}
]	
`

	JSON_STRING_VALUE_ARR = `
[
    {
        "name": "authority_url",
        "content": [
            {
                "reference": "string",
                "data": "bola"
            }
        ]
    },
    {
        "name": "credentials_type",
        "description": "Authority definition for discovery",
        "content": [
            {
                "reference": "Kubernetes token",
                "data": "kubernetes"
            }
        ]
    }
]
`

	CERTIFICATE_SIGN_REQUEST_DTO = `
{
	"request": "test",
	"format": "pkcs10",
	"raProfileAttributes": [
		{
        	"name": "authority_url",
        	"content": [
            	{
                	"reference": "string",
                	"data": "test"
            	}
        	]
    	},
	],
}
`
)

func TestUnmarshalAndCompareCertificateRequestForamt(t *testing.T) {
	certificateSignRequestDto := CertificateSignRequestDto{}

	certificateSignRequestDto.Unmarshal([]byte(CERTIFICATE_SIGN_REQUEST_DTO))

	if err := AssertCertificateSignRequestDtoRequired(certificateSignRequestDto); err != nil {
		t.Fatalf("Error asserting required fields")
	}

	if err := AssertCertificateSignRequestDtoConstraints(certificateSignRequestDto); err != nil {
		t.Fatalf("Error asserting constraints")
	}

	if certificateSignRequestDto.CertificateRequestFormat != CERTIFICATEREQUESTFORMAT_PKCS10 {
		t.Fatalf("Error asserting CertificateRequestFormat")
	}

}
