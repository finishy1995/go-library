package storage

import (
	"github.com/finishy1995/go-library/storage/src/dynamodb"
	"github.com/finishy1995/go-library/storage/src/memory"
	"github.com/finishy1995/go-library/storage/src/mongodb"
)

// Storage 存储
type Storage interface {
	// CreateTable 创建一个新的存储对象表
	// 业务正常代码不用调用这个方法，请在测试时（例如单元测试写一个测试函数来创建）
	CreateTable(value interface{}, tableName string) error

	// Create 创建一个新的存储对象（单主键时主键不相同，主键+排序键时有一个不相同）
	// value 为符合 tag 定义的 struct
	Create(value interface{}, tableName string) error

	// Delete 删除一个存储对象（单主键时不需要额外参数，主键+排序键时需要把排序键的值作为额外参数）
	// value 为符合 tag 定义的 struct
	Delete(value interface{}, tableName string, hash interface{}, args ...interface{}) error

	// Save 保存一个存储对象（请勿用这个方法创建对象，可能会造成同步性问题）
	// value 为符合 tag 定义的 struct ptr（注：一定要是 struct ptr）
	Save(value interface{}, tableName string) error

	// First 获取符合要求的存储对象（单主键时不需要额外参数，主键+排序键时需要把排序键的值作为额外参数）
	// value 为符合 tag 定义的 struct ptr
	First(value interface{}, tableName string, hash interface{}, args ...interface{}) error

	// Find 获取所有符合要求的对象，性能远低于 First，请慎重使用
	// value 为符合 tag 定义的 struct slice ptr （注：&[]struct）
	// limit 为限制数量， <= 0 即不限制数量
	// expr 为表达式（空代表不使用表达式）
	// 其他为补充表达式的具体值
	Find(value interface{}, tableName string, limit int64, expr string, args ...interface{}) error
}

var (
	typeStrMap = map[Type]string{
		InMemory: InMemoryStr,
		DynamoDB: DynamoDBStr,
		MongoDB:  MongoDBStr,
	}
)

func GetStorageTypeStr(typ Type) string {
	if str, ok := typeStrMap[typ]; ok {
		return str
	}
	return typeStrMap[InMemory]
}

func NewStorage(config *Config) Storage {
	switch config.StorageType {
	case typeStrMap[DynamoDB]:
		if config.Database != "" {
			return dynamodb.NewStorage(config.Region, config.Endpoint, config.Database, config.User, config.Password)
		}
		return dynamodb.NewStorage(config.Region, config.Endpoint, "", config.User, config.Password)
	case typeStrMap[MongoDB]:
		return mongodb.NewStorage(config.Endpoint, config.User, config.Password, config.Database)
	default:
		return memory.NewStorage(config.MaxLength, config.Tick)
	}
}
