package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// 测试图片下载和验证逻辑
	testImage := "https://www.google.com/images/branding/googlelogo/2x/googlelogo_light_color_272x92dp.png"
	fmt.Println("Testing image download and validation...")

	// 下载测试
	resp, err := http.Get(testImage)
	if err != nil {
		fmt.Printf("Download error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 检查 ContentLength
	fmt.Printf("Content length: %d\n", resp.ContentLength)

	// 保存到临时文件
	tempDir := "./download_tmp"
	os.MkdirAll(tempDir, 0755)
	tempFile := filepath.Join(tempDir, "test_image.png")

	file, err := os.Create(tempFile)
	if err != nil {
		fmt.Printf("Create file error: %v\n", err)
		return
	}
	defer file.Close()

	// 复制内容
	var downloadedSize int64
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			file.Write(buf[:n])
			downloadedSize += int64(n)
		}
		if err != nil {
			break
		}
	}

	file.Close()

	fmt.Printf("Downloaded size: %d\n", downloadedSize)

	// 验证图片完整性
	file, err = os.Open(tempFile)
	if err != nil {
		fmt.Printf("Open file error: %v\n", err)
		return
	}
	defer file.Close()

	stat, _ := file.Stat()
	fmt.Printf("File size on disk: %d\n", stat.Size())

	// 尝试解码图片
	_, _, err = image.Decode(file)
	if err != nil {
		fmt.Printf("Image decode error: %v\n", err)
	} else {
		fmt.Println("Image is valid!")
	}

	// 清理
	os.Remove(tempFile)
}
