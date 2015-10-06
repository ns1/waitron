package main

// import (
// 	"log"
// )

type Pixie struct {
	Kernel string `json:"kernel"`
	Initrd string `json:"initrd"`
}

func pixieInit(config Config) Pixie {
	var p Pixie

	//p.Kernel = "http://aptly.videoplaza.org/ubuntu/dists/trusty/main/installer-amd64/current/images/cdrom/vmlinuz"
	//p.Initrd = "http://aptly.videoplaza.org/ubuntu/dists/trusty/main/installer-amd64/current/images/cdrom/initrd.gz"
	p.Kernel = config.ImageURL + config.Kernel
	p.Initrd = config.ImageURL + config.Initrd

	return p
}
