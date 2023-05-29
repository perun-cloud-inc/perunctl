package main

import (
	perun_services "main.go/services"
	"main.go/utils"
)

func main() {

	utils.Logger = utils.GetLogger(true, "", "eventslog")
	synchronizer := perun_services.DockerSynchronizationService{}

	synchronizer.Listen()
}
