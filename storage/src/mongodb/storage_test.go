package mongodb

import (
	"fmt"
	"github.com/finishy1995/go-library/storage/core"
	"testing"
)

type Student struct {
	core.Model
	Id       string `dynamo:",hash"`
	Age      int    `json:",omitempty"`
	Score    float32
	Star     bool
	KeyValue map[string]string
	Array    []int32
	Friends  []*Student
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
	if st == nil {
		t.Errorf("st cannot be nil")
	}
}

func TestStorage_CreateTable(t *testing.T) {
	initSt()
	err := st.CreateTable(Student{})
	if err != nil {
		t.Errorf("TestStorage_CreateTable failed %s", err.Error())
	}
}

func TestStorage_Create(t *testing.T) {
	initSt()
	err := st.Create(Student{
		Id:    "111",
		Age:   1,
		Score: 0.2,
		Star:  true,
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
	})

	if err != nil {
		t.Errorf("TestStorage_CreateTable failed %s", err.Error())
	}
}

func TestStorage_Delete(t *testing.T) {
	initSt()
	err := st.Delete(Student{}, "111")
	if err != nil {
		t.Errorf("TestStorage_CreateTable failed %s", err.Error())
	}
}

func TestStorage_Save(t *testing.T) {
	initSt()
	err := st.Save(&Student{
		Id:    "111",
		Age:   1,
		Score: 0.31,
		Star:  true,
		KeyValue: map[string]string{
			"a": "1",
			"b": "3",
		},
		Array: []int32{1, 8},
	})
	if err != nil {
		t.Errorf("TestStorage_CreateTable failed %s", err.Error())
	}
}

func TestStorage_First(t *testing.T) {
	initSt()
	stu := &Student{}
	err := st.First(stu, "111")

	if err != nil {
		t.Errorf("TestStorage_CreateTable failed %s", err.Error())
	}
	fmt.Println(stu)
}

func TestStorage_Find(t *testing.T) {
	initSt()

	// 插入测试数据
	setupTestData(t)
	defer teardownTestData()

	// 测试用例
	testCases := []struct {
		name     string
		expr     string
		args     []interface{}
		expected int // 预期返回的记录数
	}{
		{
			name:     "Test AND condition",
			expr:     "Age >= ? AND Star = ?",
			args:     []interface{}{20, true},
			expected: 1, // 根据您的数据库内容调整
		},
		{
			name:     "Test OR condition",
			expr:     "Age = ? OR Score > ?",
			args:     []interface{}{18, 0.5},
			expected: 1, // 根据您的数据库内容调整
		},
		// 添加更多测试用例...
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var results []Student
			err := st.Find(&results, 10, tc.expr, tc.args...)
			if err != nil {
				t.Fatalf("Find failed: %v", err)
			}

			if len(results) != tc.expected {
				t.Errorf("Expected %d results, got %d", tc.expected, len(results))
			}

			fmt.Println(results[0])
		})
	}
}

func setupTestData(t *testing.T) {
	// 插入一些学生记录用于测试
	students := []Student{
		{Id: "1", Age: 20, Star: true /* ...其他字段... */},
		{Id: "2", Age: 18, Score: 0.75 /* ...其他字段... */},
		// ...其他测试数据...
	}

	for _, student := range students {
		err := st.Create(student)
		if err != nil {
			t.Logf("Warning: failed to insert test data: %v", err.Error())
		}
	}
}

func teardownTestData() {
	// TODO:
}
