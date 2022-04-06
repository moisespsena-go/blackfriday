package table_header

import (
	"os"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	f, _ := os.Create("teste.html")
	defer f.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
<style type="text/css">
table, th, td {
  border: 1px solid black;
  border-collapse: collapse;
}
</style>
</head>
<body>
`)
	root, err := Parse(`{a[],b:{c,d:{d1,d2:d3,d4},e},f,g:{h,i}}`)
	if err != nil {
		panic(err)
	}
	tt := root.TableHeader()
	tt.ToHTML(f)

	f.WriteString(`</body></html>`)
	return
	tests := []struct {
		name    string
		input   string
		wantTh  Node
		wantErr bool
	}{
		{"", `{a,Itens:{c,d}}`, Node{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTh, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTh, tt.wantTh) {
				t.Errorf("Parse() gotTh = %v, want %v", gotTh, tt.wantTh)
			}
		})
	}
}
