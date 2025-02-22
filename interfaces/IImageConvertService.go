package interfaces

const ImageConvertServiceID ServiceID = "ImageConvert"

type IImageConvertService interface {
	ConvertHEIC(input, output string) error
}
