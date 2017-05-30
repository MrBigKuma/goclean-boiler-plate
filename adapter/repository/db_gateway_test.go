package repository

import (
	"reflect"
	"time"
)

type MockDbGateway struct {
	ModifiedParam1 interface{}
	Result1        interface{}
	Result2        interface{}
}

func (m MockDbGateway) Get(receiverObjPtr CommonModel, id string) error {
	if m.ModifiedParam1 != nil {
		decodeVal(receiverObjPtr, m.ModifiedParam1)
	}

	if m.Result1 != nil {
		return m.Result1.(error)
	}

	return nil
}

func (m MockDbGateway) Create(dataObjPtr CommonModel) (string, error) {
	var id string
	var err error
	if m.Result1 == nil {
		id = ""
	} else {
		id = m.Result1.(string)
	}

	if m.Result2 == nil {
		err = nil
	} else {
		err = m.Result2.(error)
	}

	return id, err
}

func (m MockDbGateway) GetList(receiverObjs interface{}, index string, val interface{}) error {
	return m.Result1.(error)
}

func (m MockDbGateway) GetPartOfTable(receiverObjs interface{}, timeIndex time.Time, size int, filterMap map[string][]string) error {
	if m.ModifiedParam1 != nil {
		decodeVal(receiverObjs, m.ModifiedParam1)
	}
	if m.Result1 != nil {
		return m.Result1.(error)
	}
	return nil
}

func (m MockDbGateway) Update(receiverObjsPtr CommonModel, id string) error {
	return m.Result1.(error)
}

func (m MockDbGateway) Delete(id string) error {
	return m.Result1.(error)
}

func decodeVal(dstPtr interface{}, srcPtr interface{}) {
	dstVal := reflect.ValueOf(dstPtr).Elem()
	dstVal.Set(reflect.ValueOf(srcPtr).Elem())
}
