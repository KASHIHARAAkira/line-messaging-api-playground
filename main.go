/**
** Reference) https://github.com/go-gorp/gorp
**/

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"

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

	// LINE Messaging API Access Tokenの取得
	accessToken, err := fetchAccessToken()
	fmt.Println("🔑")
	fmt.Println(accessToken)

	// DbMapの初期化
	dbmap := initDb()
	defer dbmap.Db.Close()

	// 既に存在しているレコードを削除
	err = dbmap.TruncateTables()
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

	e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		fmt.Println(string(reqBody))
	}))

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

	e.POST("/webhook", func(c echo.Context) error {

		return c.JSON(http.StatusOK, "")
	})

	// Listen port
	e.Logger.Fatal(e.Start(":" + portNumber))
}

func fetchAccessToken() (InfoToken, error) {

	godotenv.Load(".env") // 環境変数ファイルの読み込み

	// 秘密鍵のファイルを開く
	file, err := os.Open(os.Getenv("KEYPATH"))
	if err != nil {
		log.Fatal(err)
		return InfoToken{}, err
	}
	defer file.Close()

	// ファイルから秘密鍵の読み込み
	b, err := ioutil.ReadAll(file)
	privateKey, err := jwk.ParseKey(b)
	if err != nil {
		log.Fatal(err)
		return InfoToken{}, err
	}

	// audプロパティが配列なので、aud変数を作成
	var aud []string
	aud = append(aud, "https://api.line.me/") // audプロパティの値を追加

	// JWTを構成する
	jwtexp, err := strconv.Atoi(os.Getenv("EXPJWT")) // JWTの有効期限を.envファイルから取得
	if err != nil {
		log.Fatal(err)
		return InfoToken{}, err
	}

	jwTokenSpec, err := jwt.NewBuilder().
		Subject(os.Getenv("CHID")).                                      // subプロパティにチャネルIDを入れる
		Issuer(os.Getenv("CHID")).                                       // issプロパティにチャネルIDを入れる
		Audience(aud).                                                   // audプロパティに先程作ったaudの値を入れる
		Expiration(time.Now().Add(time.Duration(jwtexp) * time.Minute)). // expプロパティにJWTの有効期間を入れる
		Build()
	if err != nil {
		log.Fatal(err)
		return InfoToken{}, err
	}

	accessTokenExp, err := strconv.Atoi(os.Getenv("EXPACC")) // アクセストークンの有効期限（日にち）を取得
	jwTokenSpec.Set("token_exp", 60*60*24*accessTokenExp)    // アクセストークンの有効期限を設定

	// JWTを発行する
	jwToken, err := jwt.Sign(jwTokenSpec, jwt.WithKey(jwa.RS256, privateKey))
	if err != nil {
		log.Fatal(err)
		return InfoToken{}, err
	}

	// チャネルアクセストークンv2.1を発行するリクエストの作成
	// 参考）https://developers.line.biz/ja/reference/messaging-api/#issue-channel-access-token-v2-1
	paraReqAccessToken := url.Values{}
	paraReqAccessToken.Set("grant_type", "client_credentials")
	paraReqAccessToken.Add("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	paraReqAccessToken.Add("client_assertion", string(jwToken))

	reqBody := strings.NewReader(paraReqAccessToken.Encode()) // リクエストのボディを作成

	// リクエストの作成
	req, err := http.NewRequest(http.MethodPost, "https://api.line.me/oauth2/v2.1/token", reqBody)
	if err != nil {
		log.Fatal(err)
		return InfoToken{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // リクエストのヘッダーを追加

	// 作成したリクエストの送信
	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
		return InfoToken{}, err
	}
	defer res.Body.Close()

	// レスポンスの解析
	var r io.Reader = res.Body

	var infoToken InfoToken
	err = json.NewDecoder(r).Decode(&infoToken)
	if err != nil {
		log.Fatal(err)
		return InfoToken{}, err
	}

	return infoToken, nil

}

type InfoToken struct {
	Token string `json:"access_token"`
	Type  string `json:"token_type"`
	Exp   int64  `json:"expires_in"`
	Id    string `json:"key_id"`
}
