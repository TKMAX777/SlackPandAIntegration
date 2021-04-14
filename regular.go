package main

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

type ReglarFile struct {
	Time      time.Time
	ChannelID string
}

type ReglarFiles struct {
	List []ReglarFile
	Path string
}

func (r *ReglarFiles) Remove(num int) error {
	switch {
	case len(r.List) == num+1:
		r.List = r.List[:num]
	case len(r.List) > num+1:
		r.List = append(r.List[:num], r.List[num+1:]...)
	default:
	}

	return r.Save()
}

func (r *ReglarFiles) Add(add ReglarFile) error {
	r.List = append(r.List, add)
	return r.Save()
}

func (r *ReglarFiles) Read(filePath string) (err error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		b = []byte("[]")
		ioutil.WriteFile(filePath, b, 0777)
	}

	err = json.Unmarshal(b, &r.List)

	r.Path = filePath

	return
}

func (r *ReglarFiles) Save() (err error) {
	b, err := json.MarshalIndent(r.List, "", "    ")
	if err != nil {
		return
	}

	err = ioutil.WriteFile(r.Path, b, 0777)
	return
}
