package domain

import "os"

type JSONParser struct {
	ID   int
	File *os.File
}
