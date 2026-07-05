package main

import "unsafe"

func main() {
	num := "Asadfasd"
	print(unsafe.Sizeof(num))
}
