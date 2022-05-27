package application

import (
	"testing"

	"github.com/omnibuildplatform/omni-repository/app"
)

func Test_DownloadImage(t *testing.T) {
	app.Bootstrap("../config", "master", "abb1b63f0c6195f5dea8fb7768b6fb581b79826e", "22.05.26-17:42:21")
	app.InitDB()
	var image app.Images
	image.ID = 20
	image.SourceUrl = "https://repo.test.osinfra.cn/data/browse/openEuler-21.03/2022-04-19/openEuler-b694e4f2-bfa8-11ec-bb72-02550a0a009d.iso"
	image.Checksum = "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"
	// downLoadImages(&image, "c:/var/E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855.iso")

}
