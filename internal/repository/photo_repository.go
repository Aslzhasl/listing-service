package repository

import (
	_ "fmt"
	"io"
	"mime/multipart"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

type PhotoRepository struct {
	DB *mongo.Database
}

func NewPhotoRepository(client *mongo.Client, dbName string) *PhotoRepository {
	return &PhotoRepository{DB: client.Database(dbName)}
}

func (r *PhotoRepository) UploadPhoto(file multipart.File, filename string) (string, error) {

	bucket, err := gridfs.NewBucket(r.DB)
	if err != nil {
		return "", err
	}

	stream, err := bucket.OpenUploadStream(filename)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	if _, err := io.Copy(stream, file); err != nil {
		return "", err
	}

	return stream.FileID.(primitive.ObjectID).Hex(), nil
}
func (r *PhotoRepository) DownloadPhoto(photoID string) ([]byte, string, error) {
	bucket, err := gridfs.NewBucket(r.DB)
	if err != nil {
		return nil, "", err
	}

	objID, err := primitive.ObjectIDFromHex(photoID)
	if err != nil {
		return nil, "", err
	}

	stream, err := bucket.OpenDownloadStream(objID)
	if err != nil {
		return nil, "", err
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, "", err
	}

	// Просто возвращаем фиксированное имя файла
	return data, "photo.jpg", nil
}
