package main

type Pixie struct {
	Kernel string `json:"kernel"`
	Initrd string `json:"initrd"`
}

func pixieInit(config Config) Pixie {
	var p Pixie

	p.Kernel = config.ImageURL + config.Kernel
	p.Initrd = config.ImageURL + config.Initrd

	return p
}
