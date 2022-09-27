package main

import (
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

// ダミーのデータとして返す、JSONの定義
type Car struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Year int64  `json:"year"`
}

func main() {
	// 環境変数ファイルの読み込み
	godotenv.Load(".env")

	// PORT番号の読み込み
	portNumber := os.Getenv("PORT")

	e := echo.New()

	// トップページに静的ページを表示
	e.Static("/", "static/")

	// 車の一覧を返す
	e.GET("/api/cars", func(c echo.Context) error {
		cars := map[string]Car{
			"1": {
				ID:   1,
				Name: "ヤリス",
				Year: 2020,
			},
			"2": {
				ID:   2,
				Name: "キャストスタイル",
				Year: 2020,
			},
		}
		return c.JSON(http.StatusOK, cars)
	})

	// 登録したい車の情報を返す
	e.POST("/api/cars", func(c echo.Context) error {
		var car Car
		if err := c.Bind(&car); err != nil {
			c.Logger().Error("Bind: ", err)
			return c.String(http.StatusBadRequest, "Bind: "+err.Error())
		}
		car.ID = 3
		return c.JSON(http.StatusCreated, car)
	})

	// Listen port
	e.Logger.Fatal(e.Start(":" + portNumber))
}
