/**
** Reference) https://github.com/go-gorp/gorp
**/

package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/go-gorp/gorp"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	_ "github.com/mattn/go-sqlite3"
)

// ダミーのデータとして返す、JSONの定義
type Car struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
	Year int64  `json:"year" db:"year"`
}

// SQLite3のデーターベースにテーブルとデータを作成
func initDb() *gorp.DbMap {
	db, err := sql.Open("sqlite3", "./cars.db") // データベースの接続
	if err != nil {
		log.Fatal(err)
	}

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}} // gorp DbMapのコンストラクタ

	dbmap.AddTableWithName(Car{}, "cars").SetKeys(true, "ID") // テーブルの追加

	err = dbmap.CreateTablesIfNotExists() // テーブルが存在していなければテーブルの追加
	if err != nil {
		log.Fatal(err)
	}

	return dbmap
}

func newCar(carName string, year int64) Car {
	return Car{
		Name: carName,
		Year: year,
	}
}

func main() {
	// 環境変数ファイルの読み込み
	godotenv.Load(".env")

	// PORT番号の読み込み
	portNumber := os.Getenv("PORT")

	// DbMapの初期化
	dbmap := initDb()
	defer dbmap.Db.Close()

	// 既に存在しているレコードを削除
	err := dbmap.TruncateTables()
	if err != nil {
		log.Fatal(err)
	}

	// いくつか車情報を追加
	car1 := newCar("ヤリス", 2020)
	car2 := newCar("キャストスタイル", 2020)
	car3 := newCar("フィット", 2019)

	// レコードを挿入
	err = dbmap.Insert(&car1, &car2, &car3)
	if err != nil {
		log.Fatal(err)
	}

	e := echo.New()

	// トップページに静的ページを表示
	e.Static("/", "static/")

	// 車の一覧を返す
	e.GET("/api/cars", func(c echo.Context) error {
		var cars []Car
		_, err = dbmap.Select(&cars, "select * from cars")
		if err != nil {
			log.Fatal(err)
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
