package mongodb

import (
	"context"
	"errors"
	"fmt"
	"github.com/finishy1995/go-library/log"
	"github.com/finishy1995/go-library/storage/core"
	"github.com/finishy1995/go-library/storage/src/tools"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"time"
)

type Storage struct {
	db *mongo.Database
}

var (
	defaultTimeout = 10 * time.Second
)

func NewStorage(endpoint, username, password, database string) *Storage {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	uri := endpoint
	if !strings.HasPrefix(uri, "mongodb://") && !strings.HasPrefix(uri, "mongodb+srv://") {
		if username != "" && password != "" {
			uri = fmt.Sprintf("mongodb://%s:%s@%s", username, password, endpoint)
		} else {
			uri = fmt.Sprintf("mongodb://%s", endpoint)
		}
	} else {
		if username != "" && password != "" {
			// 找到协议和剩余部分之间的分割点
			scheme := ""
			if strings.HasPrefix(uri, "mongodb://") {
				scheme = "mongodb://"
			} else if strings.HasPrefix(uri, "mongodb+srv://") {
				scheme = "mongodb+srv://"
			}

			if scheme != "" {
				// 分割 URI
				splitPoint := len(scheme)
				beforeURI := uri[:splitPoint]
				afterURI := uri[splitPoint:]

				// 拼接用户名和密码
				credentials := fmt.Sprintf("%s:%s@", username, password)

				// 重建完整的 URI
				uri = beforeURI + credentials + afterURI
			}
		}
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Error("try mongo connect failed, error: %s", err.Error())
		return nil
	}
	if database == "" {
		database = "data"
	}
	db := client.Database(database)
	if db == nil {
		return nil
	}
	return &Storage{db: db}
}

func (s *Storage) CreateTable(value interface{}) error {
	tableName := tools.GetStructOnlyName(value)
	if tableName == "" {
		return core.ErrUnsupportedValueType
	}
	hashKey, rangeKey := tools.GetHashAndRangeKey(value, true)
	if hashKey == "" {
		return core.ErrUnsupportedValueType
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	// 创建集合
	if err := s.db.CreateCollection(ctx, tableName); err != nil {
		return err
	}

	// 获取集合的索引视图
	collection := s.db.Collection(tableName)
	indexes := collection.Indexes()

	// 定义索引模型
	var indexModel mongo.IndexModel
	if rangeKey != "" {
		indexModel = mongo.IndexModel{
			Keys: bson.M{
				hashKey:  1, // 1 表示升序
				rangeKey: 1, // 1 表示升序
			},
			Options: options.Index().SetUnique(true),
		}
	} else {
		indexModel = mongo.IndexModel{
			Keys: bson.M{
				hashKey: 1, // 1 表示升序
			},
			Options: options.Index().SetUnique(true),
		}
	}

	// 创建索引
	_, err := indexes.CreateOne(ctx, indexModel)
	return err
}

func (s *Storage) Create(value interface{}) error {
	tableName := tools.GetStructOnlyName(value)
	if tableName == "" {
		return core.ErrUnsupportedValueType
	}
	collection := s.db.Collection(tableName)
	if collection == nil {
		return core.ErrUnsupportedValueType
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	_, err := collection.InsertOne(ctx, value)
	return err
}

func (s *Storage) Delete(value interface{}, hash interface{}, args ...interface{}) error {
	tableName := tools.GetStructOnlyName(value)
	if tableName == "" {
		return core.ErrUnsupportedValueType
	}
	hashKey, rangeKey := tools.GetHashAndRangeKey(value, true)
	if hashKey == "" {
		return core.ErrUnsupportedValueType
	}
	collection := s.db.Collection(tableName)
	if collection == nil {
		return core.ErrUnsupportedValueType
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	var err error
	if rangeKey != "" {
		if len(args) == 0 {
			return core.ErrMissingRangeValue
		}
		_, err = collection.DeleteOne(ctx, bson.D{{hashKey, hash}, {rangeKey, args[0]}})
	} else {
		_, err = collection.DeleteOne(ctx, bson.D{{hashKey, hash}})
	}
	return err
}

func (s *Storage) Save(value interface{}) error {
	tableName := tools.GetStructName(value)
	if tableName == "" {
		return core.ErrUnsupportedValueType
	}
	hashKey, rangeKey := tools.GetHashAndRangeKey(value, true)
	if hashKey == "" {
		return core.ErrUnsupportedValueType
	}
	hashValue, rangeValue := tools.GetHashAndRangeValue(value)
	if hashValue == nil {
		return core.ErrUnsupportedValueType
	}
	collection := s.db.Collection(tableName)
	if collection == nil {
		return core.ErrUnsupportedValueType
	}
	version, err := tools.TrySetStructVersion(value)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	var filter bson.D
	if rangeKey != "" {
		filter = bson.D{{hashKey, hashValue}, {rangeKey, rangeValue}, {"model.version", version}}
	} else {
		filter = bson.D{{hashKey, hashValue}, {"model.version", version}}
	}

	fields := tools.GetFieldInfo(value)
	dict := bson.D{}
	for key, val := range fields {
		dict = append(dict, bson.E{Key: key, Value: val})
	}
	update := bson.D{{"$set", dict}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return core.ErrExpiredValue
	}
	return err
}

func (s *Storage) First(value interface{}, hash interface{}, args ...interface{}) error {
	tableName := tools.GetStructName(value)
	if tableName == "" {
		return core.ErrUnsupportedValueType
	}
	hashKey, rangeKey := tools.GetHashAndRangeKey(value, true)
	if hashKey == "" {
		return core.ErrUnsupportedValueType
	}
	collection := s.db.Collection(tableName)
	if collection == nil {
		return core.ErrUnsupportedValueType
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	var filter bson.D
	if rangeKey != "" {
		if len(args) == 0 {
			return core.ErrMissingRangeValue
		}
		filter = bson.D{{hashKey, hash}, {rangeKey, args[0]}}
	} else {
		filter = bson.D{{hashKey, hash}}
	}

	result := collection.FindOne(ctx, filter)
	err := result.Decode(value)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return core.ErrNotFound
	}
	return err
}

func (s *Storage) Find(value interface{}, limit int64, expr string, args ...interface{}) error {
	tableName := tools.GetSliceStructName(value)
	if tableName == "" {
		return core.ErrUnsupportedValueType
	}
	collection := s.db.Collection(tableName)
	if collection == nil {
		return core.ErrUnsupportedValueType
	}

	// 解析表达式获取根节点
	rootNode, err := getRootNode(expr, args...)
	if err != nil {
		return err
	}
	// 根据 AST 节点构建 MongoDB 查询条件
	filter, err := buildFilterFromAST(rootNode)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	// 执行查询
	opts := options.Find().SetLimit(limit)
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	// 解码查询结果
	return cursor.All(ctx, value)
}
