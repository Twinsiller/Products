package repository

import (
	"context"
	"fmt"

	"Products_backend/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const productImagesCollection = "products"

// ProductImageDoc — документ изображения товара в MongoDB.
type ProductImageDoc struct {
	ProductID   int64  `bson:"product_id"`
	ContentType string `bson:"content_type"`
	Data        []byte `bson:"data"`
}

// SaveProductImage сохраняет или обновляет изображение товара (по product_id).
func SaveProductImage(ctx context.Context, productID int64, contentType string, data []byte) error {
	if database.DbMongo == nil {
		return fmt.Errorf("mongodb not connected")
	}
	coll := database.DbMongo.Database(database.MongoDatabaseName).Collection(productImagesCollection)
	doc := ProductImageDoc{
		ProductID:   productID,
		ContentType: contentType,
		Data:        data,
	}
	filter := bson.M{"product_id": productID}
	opts := options.Replace().SetUpsert(true)
	_, err := coll.ReplaceOne(ctx, filter, doc, opts)
	return err
}

// GetProductImage возвращает изображение товара из MongoDB. Если не найдено — (nil, nil, nil).
func GetProductImage(ctx context.Context, productID int64) (*ProductImageDoc, error) {
	if database.DbMongo == nil {
		return nil, fmt.Errorf("mongodb not connected")
	}
	coll := database.DbMongo.Database(database.MongoDatabaseName).Collection(productImagesCollection)
	var doc ProductImageDoc
	err := coll.FindOne(ctx, bson.M{"product_id": productID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// DeleteProductImage удаляет изображение товара из MongoDB.
func DeleteProductImage(ctx context.Context, productID int64) error {
	if database.DbMongo == nil {
		return fmt.Errorf("mongodb not connected")
	}
	coll := database.DbMongo.Database(database.MongoDatabaseName).Collection(productImagesCollection)
	_, err := coll.DeleteOne(ctx, bson.M{"product_id": productID})
	return err
}

// EnsureIndex создаёт индекс по product_id (опционально, для быстрого поиска).
func EnsureProductImageIndex(ctx context.Context) error {
	if database.DbMongo == nil {
		return nil
	}
	coll := database.DbMongo.Database(database.MongoDatabaseName).Collection(productImagesCollection)
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "product_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}
