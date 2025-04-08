package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	// Создаем 100x100 белое изображение
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Заливаем изображение цветом (например, светло-синим)
	fillColor := color.RGBA{R: 100, G: 150, B: 255, A: 255}
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, fillColor)
		}
	}

	// Создаем файл
	file, err := os.Create("C:/Users/user/Pictures/image.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Сохраняем изображение в формате PNG
	err = png.Encode(file, img)
	if err != nil {
		panic(err)
	}

	println("Изображение создано: C:/Users/user/Pictures/image.png")
}
