package server

import (
	"fmt"
	"os"
)

func Init() {
	r := NewRouter()

	listen := fmt.Sprintf("%s:%s", os.Getenv("HOST"), os.Getenv("PORT"))
	_ = r.Run(listen)
}
