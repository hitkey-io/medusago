package http

import "testing"

func TestParseQS(t *testing.T) {
	cases := map[string]string{
		"fields=id%2Cname&limit=1000":             `{"fields":"id,name","limit":"1000"}`,
		"id=a&id=b":                               `{"id":["a","b"]}`,
		"id[]=a&id[]=b":                           `{"id":["a","b"]}`,
		"created_at[$gte]=2020&created_at[$lt]=x": `{"created_at":{"$gte":"2020","$lt":"x"}}`,
		"a[1]=b&a[0]=c":                           `{"a":["c","b"]}`,
		"a[21]=b":                                 `{"a":{"21":"b"}}`,
		"$and[0][q]=x&$and[1][id]=y":              `{"$and":[{"q":"x"},{"id":"y"}]}`,
		"a=b&a[c]=d":                              `{"a":["b",{"c":"d"}]}`,
		"q=a+b":                                   `{"q":"a b"}`,
		"flag":                                    `{"flag":""}`,
		"a[b][c][d][e][f][g]=1":                   `{"a":{"b":{"c":{"d":{"e":{"f":{"[g]":"1"}}}}}}}`,
	}
	for in, want := range cases {
		got, _ := ParseQS(in).MarshalJSON()
		if string(got) != want {
			t.Errorf("%s: got %s want %s", in, got, want)
		}
	}
}

func TestParseQSMore(t *testing.T) {
	cases := map[string]string{
		"a[]=1&a[x]=2": `{"a":{"0":"1","x":"2"}}`,
		"a[0]=1&a=2":   `{"a":["1","2"]}`,
		"x[y]z=1":      `{"x":{"y":"1"}}`,
	}
	for in, want := range cases {
		got, _ := ParseQS(in).MarshalJSON()
		if string(got) != want {
			t.Errorf("%s: got %s want %s", in, got, want)
		}
	}
}
