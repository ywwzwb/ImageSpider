package interfaces

const ImageConvertServiceID ServiceID = "ImageConvert"

type IImageConvertService interface {
	GetFilextension() string
	Convert(input, output string) error
}
