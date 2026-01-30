package interfaces

const ImageConvertServiceID ServiceID = "ImageConvert"

type IImageConvertService interface {
	CompressImage(input, output string) error
}
