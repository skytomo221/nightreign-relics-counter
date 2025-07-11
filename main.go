package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/otiai10/gosseract/v2"
)

func main() {
	// 実行ファイルのディレクトリパスを取得
	// これにより、アプリケーションと同じ階層にあるtessdataフォルダを参照できるようになる
	ex, err := os.Executable()
	if err != nil {
		fmt.Printf("実行ファイルのパス取得エラー: %v\n", err)
		return
	}
	exPath := filepath.Dir(ex)

	// tessdataフォルダのパスを環境変数に設定
	// これにより、gosseractがtessdataの場所を認識できるようになる
	tessdataPath := filepath.Join(exPath, "tessdata")
	if err := os.Setenv("TESSDATA_PREFIX", tessdataPath); err != nil {
		fmt.Printf("環境変数の設定エラー: %v\n", err)
		return
	}
	fmt.Printf("tessdataディレクトリのパスを設定: %s\n", tessdataPath)

	// gosseractクライアントの作成
	client := gosseract.NewClient()
	// deferで必ずクライアントをクローズし、リソースを解放する
	defer client.Close()

	// ここにOCRしたい画像のパスを設定してね
	// 例: アプリケーションと同じ階層に "image.png" がある場合
	imagePath := filepath.Join(exPath, "image.png")
	// あるいは、絶対パスを指定することもできる
	// imagePath := "C:\\path\\to\\your\\image.png"

	// 画像ファイルの存在確認
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		fmt.Printf("エラー: 画像ファイルが見つかりません: %s\n", imagePath)
		fmt.Println("OCRしたい画像ファイルを指定されたパスに置いてください。")
		return
	}

	// OCR対象の画像を設定
	if err := client.SetImage(imagePath); err != nil {
		fmt.Printf("画像設定エラー: %v\n", err)
		return
	}

	// 必要に応じて認識したい言語を設定
	// 日本語を認識させたい場合は "jpn" を追加。複数の言語を指定することも可能 ("eng+jpn")
	// インストールした言語データ (.traineddata) がtessdataフォルダにあることを確認してね
	client.SetLanguage("jpn", "eng") // 例: 日本語と英語を認識

	// OCRを実行し、テキストを取得
	text, err := client.Text()
	if err != nil {
		fmt.Printf("OCR実行エラー: %v\n", err)
		return
	}

	fmt.Println("\n--- 認識されたテキスト ---")
	fmt.Println(text)
	fmt.Println("-------------------------")
}