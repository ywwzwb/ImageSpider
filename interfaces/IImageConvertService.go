package interfaces

const ImageConvertServiceID ServiceID = "ImageConvert"

type IImageConvertService interface {
	Convert(input, hash string, finishCallback func(output string, err error))
}
