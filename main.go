package main

import (
	"github.com/shoaib42/remote-move/conf"
	"github.com/shoaib42/remote-move/io"
	"github.com/shoaib42/remote-move/rest"
)

func main() {
	conf.LoadConfiguration("configuration.yaml")

	if iohelper, err := io.NewIOHelper(conf.Confs.SrcDirs, conf.Confs.DestRootDir, conf.Confs.ExcludeDirs, conf.Confs.Uid, conf.Confs.Gid); nil == err {
		if server, err := rest.New("index.html", conf.Confs.ServerBindAddr, conf.Confs.ServerBindPort, conf.Confs.AllowedCIDRs, iohelper); nil == err {
			server.Serve()
		}
	}

}
