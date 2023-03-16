package main

import "github.com/labstack/echo/v4"

func main() {
	e := echo.New()
	e.POST("/search", searchNews)
}

func searchNews(c echo.Context) error {

}
