package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/agnivade/levenshtein"
)

// --- 定数定義 ---
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

// --- 新しいデータ構造 ---
type OcrResult struct {
	Original  string
	Corrected string
}

// --- 画像処理ヘルパー関数 (変更なし) ---
func convertToGrayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayImg.Set(x, y, color.GrayModel.Convert(img.At(x, y)))
		}
	}
	return grayImg
}

func convertToBinaryAndInvert(grayImg *image.Gray, threshold uint8) *image.Gray {
	bounds := grayImg.Bounds()
	binaryImg := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if grayImg.GrayAt(x, y).Y > threshold {
				binaryImg.SetGray(x, y, color.Gray{Y: 0})
			} else {
				binaryImg.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return binaryImg
}

// --- ヘルパー関数 (変更なし) ---
func findClosestMatch(text string, candidates []string) string {
	if len(candidates) == 0 {
		return text
	}
	minDistance := -1
	bestMatch := candidates[0]
	for _, candidate := range candidates {
		distance := levenshtein.ComputeDistance(text, candidate)
		if minDistance == -1 || distance < minDistance {
			minDistance = distance
			bestMatch = candidate
		}
	}
	return bestMatch
}

func loadTsv(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("TSVファイルを開けませんでした %s: %w", filePath, err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = '\t'
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("TSVファイルの読み込みに失敗しました %s: %w", filePath, err)
	}
	var candidates []string
	for _, record := range records {
		if len(record) > 0 {
			candidates = append(candidates, record[0])
		}
	}
	return candidates, nil
}

// --- 主要な処理関数 (変更なし) ---
func openImage(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("ファイルを開けませんでした: %w", err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("画像をデコードできませんでした: %w", err)
	}
	return img, nil
}

func calculateCropRects(width, height float64) []image.Rectangle {
	cropRatios := []struct {
		x0, y0, x1, y1 float64
	}{
		{0.336, 0.523, 0.724, 0.565},
		{0.358, 0.569, 0.724, 0.625},
		{0.358, 0.625, 0.724, 0.685},
		{0.358, 0.685, 0.724, 0.745},
	}
	rects := make([]image.Rectangle, len(cropRatios))
	for i, r := range cropRatios {
		rects[i] = image.Rect(
			int(width*r.x0),
			int(height*r.y0),
			int(width*r.x1),
			int(height*r.y1),
		)
	}
	return rects
}

func preprocessRegion(img image.Image, rect image.Rectangle) *image.Gray {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	croppedImg := img.(subImager).SubImage(rect)
	grayImg := convertToGrayscale(croppedImg)
	threshold := uint8(64)
	return convertToBinaryAndInvert(grayImg, threshold)
}

func performOCR(binaryImg image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, binaryImg); err != nil {
		return "", fmt.Errorf("OCR用画像のエンコードに失敗しました: %w", err)
	}
	cmd := exec.Command("./Tesseract-OCR/tesseract.exe", "stdin", "stdout", "-l", "jpn")
	cmd.Stdin = &buf
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tesseractの実行に失敗しました: %v, 出力: %s", err, string(output))
	}
	re := regexp.MustCompile(`Estimating resolution as \d+|※適用可能な武器種のみ`)
	cleanedOutput := re.ReplaceAllString(string(output), "")
	parts := strings.Fields(cleanedOutput)
	cleanedText := strings.Join(parts, "")
	return cleanedText, nil
}

// --- 処理の統括とエントリーポイント (ロジック変更あり) ---
func processImageFile(filePath, tempDir string, region1Candidates, otherRegionsCandidates []string) {
	fmt.Printf("\n--- 処理開始: %s ---\n", filepath.Base(filePath))

	img, err := openImage(filePath)
	if err != nil {
		log.Printf("エラー: %v", err)
		return
	}

	bounds := img.Bounds()
	cropRects := calculateCropRects(float64(bounds.Dx()), float64(bounds.Dy()))

	// OCR結果（補正前・後）を保持するスライス
	results := make([]OcrResult, len(cropRects))

	for i, rect := range cropRects {
		binaryImg := preprocessRegion(img, rect)

		// (一時ファイルの保存は変更なし)
		baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		tempPath := filepath.Join(tempDir, fmt.Sprintf("%s_crop_%d.png", baseName, i+1))
		if outFile, err := os.Create(tempPath); err == nil {
			png.Encode(outFile, binaryImg)
			outFile.Close()
		}

		ocrText, err := performOCR(binaryImg)
		if err != nil {
			log.Printf("エラー: 領域 %d のOCRに失敗しました: %v", i+1, err)
			results[i] = OcrResult{Original: "OCR失敗", Corrected: "N/A"}
			continue
		}

		// 補正前と補正後の両方を保持
		results[i].Original = ocrText
		if i == 0 {
			results[i].Corrected = findClosestMatch(ocrText, region1Candidates)
		} else {
			results[i].Corrected = findClosestMatch(ocrText, otherRegionsCandidates)
		}
	}

	// 領域1の補正後テキストに基づいて色と絵文字を決定
	var colorCode, emoji string
	switch results[0].Corrected {
	case "壮大な燃える景色":
		colorCode, emoji = colorRed, "🔥"
	case "壮大な滴る景色":
		colorCode, emoji = colorBlue, "💧"
	case "壮大な輝く景色":
		colorCode, emoji = colorYellow, "✨"
	case "壮大な静まる景色":
		colorCode, emoji = colorGreen, "🍃"
	default:
		colorCode, emoji = colorReset, "❔"
	}

	// 全ての結果を新しいフォーマットで表示
	for i, res := range results {
		// 補正前と後が同じ場合は矢印を表示しない
		if res.Original == res.Corrected {
			if i == 0 {
				fmt.Printf("[領域 %d OCR結果]: %s %s%s%s\n", i+1, emoji, colorCode, res.Corrected, colorReset)
			} else {
				fmt.Printf("[領域 %d OCR結果]: %s%s%s\n", i+1, colorCode, res.Corrected, colorReset)
			}
		} else {
			if i == 0 {
				fmt.Printf("[領域 %d OCR結果]: %s %s%s%s ← %s\n", i+1, emoji, colorCode, res.Corrected, colorReset, res.Original)
			} else {
				fmt.Printf("[領域 %d OCR結果]: %s%s%s ← %s\n", i+1, colorCode, res.Corrected, colorReset, res.Original)
			}
		}
	}
}

// main関数 (変更なし)
func main() {
	region1Candidates := []string{
		"壮大な燃える景色",
		"壮大な滴る景色",
		"壮大な輝く景色",
		"壮大な静まる景色",
	}
	otherRegionsCandidates, err := loadTsv("relics.tsv")
	if err != nil {
		log.Printf("警告: relics.tsvの読み込みに失敗しました。領域2-4の補正は行われません。エラー: %v", err)
	}

	dataDir := "./data"
	tempDir := "./temp"

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("エラー: 一時ディレクトリの作成に失敗しました: %v", err)
	}

	files, err := os.ReadDir(dataDir)
	if err != nil {
		log.Fatalf("エラー: dataディレクトリの読み込みに失敗しました: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			processImageFile(filepath.Join(dataDir, fileName), tempDir, region1Candidates, otherRegionsCandidates)
		}
	}

	fmt.Println("\n--- 全ての処理が完了しました ---")
}
