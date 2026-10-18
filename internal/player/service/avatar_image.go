package service

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func makeAvatar(data []byte) ([]byte, error) {
	info, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidPhoto
	}
	if info.Width > 5000 || info.Height > 5000 {
		return nil, ErrPhotoTooLarge
	}

	photo, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidPhoto
	}

	b := photo.Bounds()
	side := min(b.Dx(), b.Dy())
	x := b.Min.X + (b.Dx()-side)/2
	y := b.Min.Y + (b.Dy()-side)/2
	square := image.Rect(x, y, x+side, y+side)

	avatar := image.NewRGBA(image.Rect(0, 0, 512, 512))
	draw.Draw(avatar, avatar.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(avatar, avatar.Bounds(), photo, square, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, avatar, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
