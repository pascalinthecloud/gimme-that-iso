package ui

import (
	_ "embed"
	"fmt"
)

//go:embed banner.txt
var bannerContent string

func PrintBanner() {
	fmt.Println(bannerContent)
}
