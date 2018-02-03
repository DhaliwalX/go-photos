// a simple photos service that allows to save photos and returns an id to access it
package photos

import (
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/minio/minio-go"
	"github.com/satori/go.uuid"
	"io"
	"io/ioutil"
	"log"
	url2 "net/url"
	"os"
	"strings"
	"time"
)

type Album struct {
	ID     uint    `json:"id"`
	Name   string  `json:"name,omitempty"`
	Photos []Photo `json:"photos"`
}

type Photo struct {
	ID      uint   `json:"id"`
	Hash    string `json:"hash"`
	AlbumId uint   `json:"album_id"`
}

type PhotoStore interface {
	Save(object interface{}) error
	FindPhoto(id uint) (*Photo, error)
	FindAlbum(name string) (*Album, error)
	GetPhotosOfAlbum(album *Album) ([]Photo, error)
}

type PhotoService struct {
	client     *minio.Client
	bucketName string
	store      PhotoStore
	logger     *log.Logger
}

func (service *PhotoService) Log(format string, rest ...interface{}) {
	service.logger.Printf(format+"\n", rest...)
}

func (service PhotoService) SavePhoto(image io.Reader, album *Album) (photo *Photo, err error) {
	bimg, err := ioutil.ReadAll(image)
	if err != nil {
		return nil, errors.New("couldn't read image")
	}
	i := strings.Index(string(bimg), ",")
	if i == -1 {
		return nil, errors.New("unrecognized image format")
	}
	decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(bimg[i+1:]))
	shaStr, err := uuid.NewV4()

	if err != nil {
		return
	}

	log.Println(shaStr)
	_, err = service.client.PutObject(service.bucketName, album.Name+"/"+shaStr.String()+".png", decoder, -1, minio.PutObjectOptions{})

	if err != nil {
		service.Log("Unable to upload image: %v", err)
		return
	}

	photo = &Photo{
		Hash: shaStr.String() + ".png",
	}

	err = service.store.Save(photo)
	return
}

func (service PhotoService) GetPhoto(album *Album, name string) (reader io.Reader, err error) {
	object, err := service.client.GetObject(service.bucketName, album.Name+"/"+name, minio.GetObjectOptions{})
	if err != nil {
		return
	}
	reader = object
	return
}

func (service PhotoService) GetSignedUrlOfImage(album *Album, name string) (url string, err error) {
	reqParams := make(url2.Values)
	//reqParams.Set("response-content-disposition", "attachment; filename=\""+name+"\"")
	object, err := service.client.PresignedGetObject(service.bucketName, album.Name+"/"+name, time.Second*60*5, reqParams)
	if err != nil {
		return
	}
	url = object.String()
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
	album := Album{Name: name}
	return service.store.Save(&album)
}

func NewPhotoService(url string, accessKey string, secretKey string, store PhotoStore, bucketName string) (PhotoService, error) {
	service := PhotoService{}
	service.store = store
	service.bucketName = bucketName
	service.logger = log.New(os.Stdout, "Photos Service", log.Llongfile)
	var err error
	service.client, err = minio.New(url, accessKey, secretKey, false)

	if err != nil {
		err = service.client.MakeBucket(bucketName, "us-east-2")
		if err != nil {
			if service.client.BucketExists(bucketName) {
				log.Printf("Bucket %s already exists!", bucketName)
			} else {
				log.Printf("Unable to make bucket %s: %v", bucketName, err)
			}
		}
	}
	return service, err
}
