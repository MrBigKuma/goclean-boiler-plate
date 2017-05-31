package rethinkdbHelper

import (
	"errors"
	"github.com/Sirupsen/logrus"
	rdb "github.com/dancannon/gorethink"
	"goclean/adapter/repository"
	"time"
)

type rdbGateway struct {
	session   *rdb.Session
	TableName string
}

func NewRdbGateway(session *rdb.Session, tableName string) repository.DbGateway {
	return rdbGateway{
		session:   session,
		TableName: tableName,
	}
}

// Get single record for a table
func (r rdbGateway) Get(receiverObjPtr repository.CommonModel, id string) error {
	resp, err := rdb.Table(r.TableName).Get(id).Run(r.session)
	if err != nil {
		return err
	}
	defer resp.Close()

	err = resp.One(receiverObjPtr)
	if err != nil {
		return err
	}

	return nil
}

// Create new object and return its id
// Return either (id, err) or ("", nil)
func (r rdbGateway) Create(dataObjPtr repository.CommonModel) (string, error) {
	now := time.Now()
	dataObjPtr.SetLastUpdated(now)
	dataObjPtr.SetCreatedTime(now)

	resp, err := rdb.Table(r.TableName).Insert(dataObjPtr).RunWrite(r.session)
	if err != nil {
		return "", err
	}

	// Unexpected error
	if len(resp.GeneratedKeys) == 0 {
		logrus.Error("Data wasn't created")
		return "", errors.New("Data wasn't created")
	}

	// TODO: update id instead of response
	return resp.GeneratedKeys[0], nil
}

// Get list of resource base on an index
func (r rdbGateway) GetList(receiverObjs interface{}, index string, val interface{}) error {
	resp, err := rdb.Table(r.TableName).
		GetAllByIndex(index, val).
		OrderBy(rdb.Desc("createdTime")).
		Run(r.session)
	if err != nil {
		return err
	}
	defer resp.Close()

	err = resp.All(receiverObjs)
	if err != nil {
		return err
	}

	return nil
}

// Get a part of table (paging) with sort by createdTime
func (r rdbGateway) GetPartOfTable(receiverObjs interface{}, timeIndex time.Time, size int, filterMap map[string][]string) error {
	// Recursive func to generate filter condition
	var termGenerator func(fs []string, k string) rdb.Term
	termGenerator = func(fs []string, k string) rdb.Term {
		if len(fs) == 1 {
			return rdb.Row.Field(k).Eq(fs[0])
		}
		return rdb.Or(rdb.Row.Field(k).Eq(fs[0]), termGenerator(fs[1:], k))
	}

	// Create filter condition
	filterConditions := []rdb.Term{}
	for key, filters := range filterMap {
		if len(filters) > 0 {
			filterConditions = append(filterConditions, termGenerator(filters, key))
		}
	}

	// Build query & run
	query := rdb.Table(r.TableName).
		Between(rdb.MinVal, timeIndex, rdb.BetweenOpts{Index: "createdTime"}).
		OrderBy(rdb.OrderByOpts{Index: rdb.Desc("createdTime")})
	for _, cond := range filterConditions {
		query = query.Filter(cond)
	}
	resp, err := query.Limit(size).Run(r.session)
	if err != nil {
		return err
	}
	defer resp.Close()

	return resp.All(receiverObjs)
}

// Update data by its id
func (r rdbGateway) Update(receiverObjsPtr repository.CommonModel, id string) error {
	receiverObjsPtr.SetLastUpdated(time.Now())
	_, err := rdb.Table(r.TableName).Get(id).Update(receiverObjsPtr).RunWrite(r.session)
	if err != nil {
		return err
	}

	return nil
}

// Get single record for a table
func (r rdbGateway) Delete(id string) error {
	_, err := rdb.Table(r.TableName).Get(id).Delete().RunWrite(r.session)
	return err
}
