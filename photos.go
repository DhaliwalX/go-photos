// a simple photos service that allows to save photos and returns an id to access it
package photos

import (
	"github.com/princedhaliwal/quadb/src/models"
	"github.com/minio/minio-go"
	"log"
	"io"
	"github.com/satori/go.uuid"
)

type Album struct {
	models.Model
	Name string `json:"name,omitempty"`
	Photos []Photo `json:"photos"`
}

type Photo struct {
	models.Model
	Hash string `json:"hash"`
	AlbumId uint `json:"album_id"`
}

type PhotoStore interface {
	Save(object interface{}) error
	FindPhoto(id uint) (*Photo, error)
	FindAlbum(name string) (*Album, error)
	GetPhotosOfAlbum(album *Album) ([]Photo, error)
}

type PhotoService struct {
	client *minio.Client
	bucketName string
	store PhotoStore
}

func (service PhotoService) SavePhoto(image io.Reader, album *Album) (photo *Photo, err error) {
	shaStr, err := uuid.NewV4()

	if err != nil {
		return
	}

	log.Println(shaStr)
	_, err = service.client.PutObject(service.bucketName, shaStr.String()+".png", image, -1, minio.PutObjectOptions{})

	if err != nil {
		return
	}

	photo = &Photo{
		Hash: shaStr.String()+".png",
	}

	err = service.store.Save(photo)
	return
}

func (service PhotoService) GetPhoto(album *Album, name string) (reader io.Reader, err error) {
	object, err := service.client.GetObject(service.bucketName, name, minio.GetObjectOptions{})
	if err != nil {
		return
	}
	reader = object
	return
}

func (service PhotoService) GetAlbum(name string) (album *Album, err error) {
	album, err = service.store.FindAlbum(name)
	return
}

func (service PhotoService) GetPhotosOfAlbum(album *Album) (photos []Photo, err error) {
	return service.store.GetPhotosOfAlbum(album)
}

func (service PhotoService) CreateAlbum(name string) (err error) {
	album := Album{ Name: name }
	return service.store.Save(&album)
}

func NewPhotoService(url string, accessKey string, secretKey string, store *PhotoStore) (PhotoService, error) {
	service := PhotoService{}
	service.store = *store
	var err error
	service.client, err = minio.New(url, accessKey, secretKey, true)
	return service, err
}
