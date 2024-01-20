package mongodb

import (
	"context"
	"github.com/finishy1995/go-library/storage/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type Student struct {
	core.Model
	Id       string `dynamo:",hash"`
	Age      int
	Score    float32 `dynamo:",default=0.1"`
	Star     bool
	KeyValue map[string]string
	Array    []int32
	Friends  []*Student
}

type TestInsertStructAgain struct {
	TestInsertStruct
}

type TestInsertStruct struct {
	Id string `dynamo:",hash"`
	core.Model
}

type Classmates struct {
	TestInsertStructAgain
	UserId int     `dynamo:",range"`
	Score  float32 `dynamo:",default=0.2"`
}

var (
	st *Storage
)

func initSt() {
	if st == nil {
		st = NewStorage("mongodb://root:123456@127.0.0.1:27017", "", "", "test")
	}
}

func TestNewStorage(t *testing.T) {
	initSt()
	assert.NotNil(t, st, "st cannot be nil")
}

func TestStorage_CreateTable(t *testing.T) {
	dropTable()

	createTable(t)
	// 重试，应该没有错误
	createTable(t)
}

func createTable(t *testing.T) {
	err := st.CreateTable(Student{})
	assert.Nil(t, err)
	err = st.CreateTable(Classmates{})
	assert.Nil(t, err)
}

func dropTable() {
	initSt()
	st.db.Collection("Student").Drop(context.Background())
	st.db.Collection("Classmates").Drop(context.Background())
}

func TestStorage_Create(t *testing.T) {
	dropTable()
	createTable(t)

	asserts := assert.New(t)
	stu := Student{
		Id:   "111",
		Star: true,
		KeyValue: map[string]string{
			"a": "1",
			"b": "2",
		},
		Array: []int32{1, 7},
		Friends: []*Student{
			{
				Id: "22",
			},
		},
	}
	var stuReal Student
	err := st.Create(stu)
	asserts.Nil(err)
	err = st.Create(stu)
	asserts.NotNil(err)
	err = st.First(&stuReal, "111")
	asserts.Nil(err)
	asserts.Equal(true, stuReal.Star)
	asserts.Equal(float32(0.1), stuReal.Score)

	cls := Classmates{
		TestInsertStructAgain: TestInsertStructAgain{TestInsertStruct{
			Id: "111",
		}},
		UserId: 10,
	}
	var clsReal Classmates
	err = st.Create(cls)
	asserts.Nil(err)
	err = st.Create(cls)
	asserts.NotNil(err)
	err = st.First(&clsReal, "111")
	asserts.Equal(core.ErrMissingRangeValue, err)
	err = st.First(&clsReal, "111", 10)
	asserts.Nil(err)
	asserts.Equal(10, clsReal.UserId)
	asserts.Equal(float32(0.2), clsReal.Score)
}

func TestStorage_Delete(t *testing.T) {
	dropTable()
	createTable(t)

	asserts := assert.New(t)
	err := st.Delete(Student{}, "111")
	asserts.Nil(err)
	err = st.Create(Student{Id: "2"})
	asserts.Nil(err)
	err = st.Delete(Student{}, "111")
	asserts.Nil(err)
}

func TestStorage_Save(t *testing.T) {
	dropTable()
	createTable(t)

	asserts := require.New(t)
	stu := &Student{
		Id:    "111",
		Age:   1,
		Score: 0.31,
		Star:  true,
		KeyValue: map[string]string{
			"a": "1",
			"b": "3",
		},
		Array: []int32{1, 8},
	}
	err := st.Save(stu)
	asserts.Equal(core.ErrExpiredValue, err)
	err = st.Create(*stu)
	asserts.Nil(err)
	asserts.Equal(uint64(1), stu.Version)
	stu.Score += 1
	err = st.Save(stu)
	asserts.Nil(err)

	var stuReal Student
	err = st.First(&stuReal, "111")
	asserts.Nil(err)
	asserts.Equal(float32(1.31), stuReal.Score)
	asserts.Equal(2, len(stuReal.KeyValue))
	asserts.Equal("1", stuReal.KeyValue["a"])
	asserts.Equal("3", stuReal.KeyValue["b"])
	asserts.Equal(2, len(stuReal.Array))
	asserts.Equal(int32(1), stuReal.Array[0])
	asserts.Equal(int32(8), stuReal.Array[1])
	asserts.Equal(uint64(2), stuReal.Version)

	cls := &Classmates{
		TestInsertStructAgain: TestInsertStructAgain{TestInsertStruct{
			Id: "2",
		}},
		UserId: 0,
	}
	err = st.Create(*cls)
	asserts.Nil(err)
	cls.UserId += 1
	err = st.Save(cls)
	asserts.Equal(core.ErrExpiredValue, err)

	// 当出现 save failed，一定要先 first，再 save
	err = st.First(cls, "2", 0)
	cls.Score++
	err = st.Save(cls)
	asserts.Nil(err)
}

func TestStorage_First(t *testing.T) {
	dropTable()
	createTable(t)

	asserts := assert.New(t)
	stu := &Student{}
	err := st.First(stu, "111")
	asserts.Equal(core.ErrNotFound, err)

	err = st.Create(Classmates{
		TestInsertStructAgain: TestInsertStructAgain{TestInsertStruct{
			Id: "33",
		}},
		UserId: 12,
		Score:  2.3,
	})
	asserts.Nil(err)
	var cls Classmates
	err = st.First(&cls, "33")
	asserts.Equal(core.ErrMissingRangeValue, err)
	err = st.First(&cls, "33", 11)
	asserts.Equal(core.ErrNotFound, err)
	err = st.First(&cls, "33", 12)
	asserts.Nil(err)
	asserts.Equal(float32(2.3), cls.Score)
}

func TestStorage_Find(t *testing.T) {
	dropTable()
	createTable(t)

	// 插入更多的测试数据
	setupTestData(t)

	// 测试用例
	testCases := []struct {
		name     string
		expr     string
		args     []interface{}
		limit    int64
		expected int // 预期返回的记录数
	}{
		{
			name:     "Test AND condition with normal limit",
			expr:     "Age >= ? AND Star = ?",
			args:     []interface{}{18, true},
			limit:    -1,
			expected: 2, // 根据您的数据库内容调整
		},
		{
			name:     "Test OR condition with no limit",
			expr:     "Age > ? OR Score > ?",
			args:     []interface{}{19, 0.5},
			limit:    -1,
			expected: 3, // 根据您的数据库内容调整
		},
		{
			name:     "Test condition with limited results",
			expr:     "Age > ?",
			args:     []interface{}{15},
			limit:    2,
			expected: 2, // 指定限制为2，预期返回2条记录
		},
		{
			name:     "Test condition with limited results",
			expr:     "Model.Version > ?",
			args:     []interface{}{1},
			limit:    2,
			expected: 0, // 指定限制为2，预期返回2条记录
		},
		{
			name:     "Test condition with no expr",
			expr:     "",
			args:     []interface{}{1},
			limit:    10,
			expected: 4,
		},
		{
			name:     "Test condition with email",
			expr:     "Id = ?",
			args:     []interface{}{"finishy@qq.com"},
			limit:    10,
			expected: 1,
		},
		// 添加更多测试用例...
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var results []Student
			err := st.Find(&results, tc.limit, tc.expr, tc.args...)
			if err != nil {
				t.Fatalf("Find failed: %v", err)
			}

			if len(results) != tc.expected {
				t.Errorf("Expected %d results, got %d for test case '%s'", tc.expected, len(results), tc.name)
			}
		})
	}
}

func setupTestData(t *testing.T) {
	// 插入更多的学生记录用于测试
	students := []Student{
		{Id: "1", Age: 20, Star: true /* ...其他字段... */},
		{Id: "2", Age: 18, Score: 0.75 /* ...其他字段... */},
		{Id: "3", Age: 19, Star: true /* ...其他字段... */},
		{Id: "finishy@qq.com", Age: 21 /* ...其他字段... */},
		// ...其他测试数据...
	}

	for _, student := range students {
		err := st.Create(student)
		if err != nil {
			t.Logf("Warning: failed to insert test data: %v", err.Error())
		}
	}
}
